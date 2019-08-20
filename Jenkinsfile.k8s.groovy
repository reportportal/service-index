#!groovy

//String podTemplateConcat = "${serviceName}-${buildNumber}-${uuid}"
def label = "worker-${UUID.randomUUID().toString()}"
println("label")
println("${label}")

podTemplate(
        label: "${label}",
        containers: [
                containerTemplate(name: 'jnlp', image: 'jenkins/jnlp-slave:alpine'),
                containerTemplate(name: 'docker', image: 'docker:dind', ttyEnabled: true, alwaysPullImage: true, privileged: true,
                        command: 'dockerd --host=unix:///var/run/docker.sock --host=tcp://0.0.0.0:2375 --storage-driver=overlay'),
                //alpine image does not have make included
                containerTemplate(name: 'golang', image: 'golang:1.12.7', ttyEnabled: true, command: 'cat'),

                containerTemplate(name: 'kubectl', image: 'lachlanevenson/k8s-kubectl:v1.8.8', command: 'cat', ttyEnabled: true),
                containerTemplate(name: 'helm', image: 'lachlanevenson/k8s-helm:latest', command: 'cat', ttyEnabled: true),
//                containerTemplate(name: 'yq', image: 'mikefarah/yq', command: 'cat', ttyEnabled: true)
                containerTemplate(name: 'jq', image: 'everpeace/curl-jq', command: 'cat', ttyEnabled: true)
        ],
        volumes: [
                emptyDirVolume(memory: false, mountPath: '/var/lib/docker'),
                secretVolume(mountPath: '/etc/.dockercreds', secretName: 'docker-creds'),
                hostPathVolume(mountPath: '/usr/local/go/pkg/mod', hostPath: '/tmp/jenkins/go')
        ]
) {

    node("${label}") {
        def srvRepo = "quay.io/reportportal/service-index"
        def srvVersion = "BUILD-${env.BUILD_NUMBER}"
        def tag = "$srvRepo:$srvVersion"

        def k8sDir = "kubernetes"
        def ciDir = "reportportal-ci"
        def appDir = "app"

        properties([
                pipelineTriggers([
                        pollSCM('H/10 * * * *')
                ])
        ])

        stage('Configure') {
            container('docker') {
                sh 'echo "Initialize environment"'
                sh """
                QUAY_USER=\$(cat "/etc/.dockercreds/username")
                cat "/etc/.dockercreds/password" | docker login -u \$QUAY_USER --password-stdin quay.io
                """
            }
            container('helm') {
                sh 'helm init --client-only'
                sh 'helm repo update'
            }
        }
        parallel 'Checkout Infra': {
            stage('Checkout Infra') {
                sh 'mkdir -p ~/.ssh'
                sh 'ssh-keyscan -t rsa github.com >> ~/.ssh/known_hosts'
                sh 'ssh-keyscan -t rsa git.epam.com >> ~/.ssh/known_hosts'
                dir(k8sDir) {
                    git branch: "master", url: 'https://github.com/reportportal/kubernetes.git'

                }
                dir(ciDir) {
                    git credentialsId: 'epm-gitlab-key', branch: "master", url: 'git@git.epam.com:epmc-tst/reportportal-ci.git'
                }

            }
        }, 'Checkout Service': {
            stage('Checkout Service') {
                dir(appDir) {
                    checkout scm
                }
            }
        }

        def test = load "${ciDir}/jenkins/scripts/test.groovy"
        def utils = load "${ciDir}/jenkins/scripts/util.groovy"

        stage('DVT Test') {
            def srvUrl = utils.getServiceUrl("reportportal", "index")
            container('jq') {
                test.checkVersion(srvUrl, "$srvVersion")
            }
        }
    }
}

