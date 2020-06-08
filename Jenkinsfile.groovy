#!groovy

node {

       load "$JENKINS_HOME/jobvars.env"

       dir('src/github.com/reportportal/service-index') {

           stage('Checkout'){
                checkout scm
            }

            docker.withServer("$DOCKER_HOST") {
                stage('Build Docker Image') {
                   sh """          
                   MAJOR_VER=\$(cat VERSION)
                   BUILD_VER="\${MAJOR_VER}-${env.BUILD_NUMBER}"
                   make build-image v=\$BUILD_VER
                   """
                }

                stage('Deploy container') {
                   sh "docker-compose -p reportportal -f $COMPOSE_FILE_RP up -d --force-recreate index"
                   stage('Push to ECR') {
                      withEnv(["AWS_URI=${AWS_URI}", "AWS_REGION=${AWS_REGION}"]) {
                             sh 'docker tag reportportal-dev/service-index ${AWS_URI}/service-index'
                             sh 'docker tag reportportal-dev/service-index ${LOCAL_REGISTRY}/service-index'
                             sh 'docker push ${LOCAL_REGISTRY}/service-index'
                             def image = env.AWS_URI + '/service-index'
                             def url = 'https://' + env.AWS_URI
                             def credentials = 'ecr:' + env.AWS_REGION + ':aws_credentials'
                             docker.withRegistry(url, credentials) {
                                docker.image(image).push('SNAPSHOT-${BUILD_NUMBER}')
                             }
                      }
                   }              
                }
                   
                stage('Cleanup') {
                   docker.withServer("$DOCKER_HOST") {
                       withEnv(["AWS_URI=${AWS_URI}", "LOCAL_REGISTRY=${LOCAL_REGISTRY}"]) {
                                  sh 'docker rmi ${AWS_URI}/service-index:SNAPSHOT-${BUILD_NUMBER}'
                                  sh 'docker rmi ${AWS_URI}/service-index:latest'
                                  sh 'docker rmi ${LOCAL_REGISTRY}/service-index:latest'
                              }
                       }
                }
            }

        }
}

