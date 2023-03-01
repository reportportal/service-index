#!groovy

node {

       load "$JENKINS_HOME/jobvars.env"

       dir('src/github.com/reportportal/service-index') {

           stage('Checkout'){
                checkout scm
            }

                stage('Build Docker Image') {
                   sh """          
                   MAJOR_VER=\$(cat VERSION)
                   BUILD_VER="\${MAJOR_VER}-${env.BUILD_NUMBER}"
                   make build-image v=\$BUILD_VER
                   """
                }

                stage('Deploy container') {
                   stage('Push to ECR') {
                      withEnv(["AWS_URI=${AWS_URI}", "AWS_REGION=${AWS_REGION}"]) {
                             sh 'docker tag reportportal-dev/service-index ${AWS_URI}/service-index:SNAPSHOT-${BUILD_NUMBER}'
                             def image = env.AWS_URI + '/service-index' + ':SNAPSHOT-' + env.BUILD_NUMBER
                             def url = 'https://' + env.AWS_URI
                             def credentials = 'ecr:' + env.AWS_REGION + ':aws_credentials'
                             echo image
                             docker.withRegistry(url, credentials) {
                                docker.image(image).push('SNAPSHOT-${BUILD_NUMBER}')
                             }
                      }
                   }              
                }
                   
                stage('Cleanup') {
                       withEnv(["AWS_URI=${AWS_URI}"]) {
                                  sh 'docker rmi ${AWS_URI}/service-index:SNAPSHOT-${BUILD_NUMBER}'
                                  sh 'docker rmi ${AWS_URI}/service-index:latest'
                       }
                }

        }
}

