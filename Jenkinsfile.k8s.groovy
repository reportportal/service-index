#!groovy
@Library('commons') _

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

        dockerUtil.init()
        helm.init()
        util.scheduleRepoPoll()

        def majorVersion;
        dir('app') {
            majorVersion = util.execStdout('cat VERSION')
        }

        def srvRepo = "quay.io/reportportal/service-index"
        def srvVersion = "$majorVersion-BUILD-${env.BUILD_NUMBER}"
        def tag = "$srvRepo:$srvVersion"

        sast('reportportal_services_sast', 'rp/carrier/config.yaml', 'service-index', false)

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
            helm.deploy("./$k8sChartDir", ["serviceindex.repository": srvRepo, "serviceindex.tag": srvVersion], false) // with wait
        }

        stage('DVT Test') {
            helm.testDeployment("reportportal", "reportportal-index", "$srvVersion", 30, 10)
        }

        dast('reportportal_dast', 'rp/carrier/config.yaml', 'rpportal_dev_dast', false)
    }
}
