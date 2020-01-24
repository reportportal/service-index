#!groovy

//String podTemplateConcat = "${serviceName}-${buildNumber}-${uuid}"
def label = "worker-${env.JOB_NAME}-${UUID.randomUUID().toString()}"
println("Worker name: ${label}")

podTemplate(
        label: "${label}",
        containers: [
                containerTemplate(name: 'jnlp', image: 'jenkins/jnlp-slave:alpine'),
                containerTemplate(name: 'docker', image: 'docker', command: 'cat', ttyEnabled: true),
                //alpine image does not have make included
                containerTemplate(name: 'golang', image: 'golang:1.12.7', ttyEnabled: true, command: 'cat'),

                containerTemplate(name: 'kubectl', image: 'lachlanevenson/k8s-kubectl:v1.8.8', command: 'cat', ttyEnabled: true),
                containerTemplate(name: 'helm', image: 'lachlanevenson/k8s-helm:v3.0.2', command: 'cat', ttyEnabled: true),
                // containerTemplate(name: 'yq', image: 'mikefarah/yq', command: 'cat', ttyEnabled: true),
                containerTemplate(name: 'httpie', image: 'blacktop/httpie', command: 'cat', ttyEnabled: true)
        ],
        volumes: [
                hostPathVolume(hostPath: '/var/run/docker.sock', mountPath: '/var/run/docker.sock'),
                secretVolume(mountPath: '/etc/.dockercreds', secretName: 'docker-creds'),
                hostPathVolume(mountPath: '/go/pkg/mod', hostPath: '/tmp/jenkins/go')
        ]
) {

    node("${label}") {
        /**
         * General ReportPortal Kubernetes Configuration and Helm Chart
         */
        def k8sDir = "kubernetes"
        def k8sChartDir = "$k8sDir/reportportal/v5"

        /**
         * Jenkins utilities and environment Specific k8s configuration
         */
        def ciDir = "reportportal-ci"
        def appDir = "app"
        def testsDir = "tests"

        parallel 'Checkout Infra': {
            stage('Checkout Infra') {
                sh 'mkdir -p ~/.ssh'
                sh 'ssh-keyscan -t rsa github.com >> ~/.ssh/known_hosts'
                sh 'ssh-keyscan -t rsa git.epam.com >> ~/.ssh/known_hosts'
                dir('kubernetes') {
                    git branch: "master", url: 'https://github.com/reportportal/kubernetes.git'

                }
                dir(ciDir) {
                    git credentialsId: 'epm-gitlab-key', branch: "master", url: 'git@git.epam.com:epmc-tst/reportportal-ci.git'
                }
            }
        }, 'Checkout Service': {
            stage('Checkout Service') {
                dir('app') {
                    checkout scm
                }
            }
        }
        def test = load "${ciDir}/jenkins/scripts/test.groovy"
        def utils = load "${ciDir}/jenkins/scripts/util.groovy"
        def helm = load "${ciDir}/jenkins/scripts/helm.groovy"
        def docker = load "${ciDir}/jenkins/scripts/docker.groovy"

        docker.init()
        helm.init()


        utils.scheduleRepoPoll()

        def majorVersion;
        dir('app') {
            majorVersion = utils.execStdout('cat VERSION')
        }

        def srvRepo = "quay.io/reportportal/service-index"
        def srvVersion = "$majorVersion-BUILD-${env.BUILD_NUMBER}"
        def tag = "$srvRepo:$srvVersion"


        // Add to the main CI pipelines SAST step:
        def sastJobName = 'reportportal_services_sast'
        stage('Run SAST') {
            catchError(buildResult: 'SUCCESS', stageResult: 'FAILURE') {
                println("Triggering build of SAST job: ${sastJobName}...")
                build job: sastJobName,
                        parameters: [
                                string(name: 'CONFIG', value: 'carrier/config.yaml'),
                                string(name: 'SUITE', value: env.JOB_NAME),
                                booleanParam(name: 'DEBUG', value: false)
                        ],
                        propagate: false, wait: false // true or false: Wait for job finish
            }
        }

        dir('app') {
            container('golang') {
                stage('Build') {
                    sh "make get-build-deps"
                    sh "make build v=$srvVersion"
                }
            }
            container('docker') {
                stage('Build Image') {
                    sh "docker build -t $tag -f DockerfileDev ."
                }
                stage('Push Image') {
                    sh "docker push $tag"
                }
            }
        }

        stage('Deploy to Dev') {
            // def valsFile = "merged.yml"
            // container('yq') {
            //     sh "yq m -x $k8sChartDir/values.yaml $ciDir/rp/values-ci.yml > $valsFile"
            // }

            helm.deploy("./$k8sChartDir", ["serviceindex.repository": srvRepo, "serviceindex.tag": srvVersion], true) // with wait
        }

        stage('DVT Test') {
            def srvUrl
            container('kubectl') {
                srvUrl = utils.getServiceEndpoint("reportportal", "reportportal-index")
            }
            if (srvUrl == null) {
                error("Unable to retrieve service URL")
            }
            container('httpie') {
                test.checkVersion("http://$srvUrl", "$srvVersion", 30, 10) // 30 attempts with 10 seconds timeout
            }
        }

        // Add to the service-ui ci pipeline DAST step:
        def dastJobName = 'reportportal_dast'
        stage('Run DAST') {
            catchError(buildResult: 'SUCCESS', stageResult: 'FAILURE') {
                println("Triggering build of SAST job: ${dastJobName}...")
                build job: dastJobName,
                        parameters: [
                                string(name: "CONFIG", value: "$ciDir/rp/carrier/config.yaml"),
                                string(name: "SUITE", value: "rpportal_dev_dast"),
                                string(name: "BASE_PATH", value: "$ciDir/rp"),
                                booleanParam(name: "DEBUG", defaultValue: false)
                        ],
                        propagate: false, wait: false // true or false: Wait for job finish
            }
        }
//        stage('Smoke Tests') {
//            def srvUrl
//            dir (testsDir) {
//                container('postman') {
//                    sh "newman run postman/service-api.postman_collection.json --env-var rp_url=https://rp.avarabyeu.me av.postman_environment.json -r cli,junit"
//                }
//            }
//        }

    }
}

