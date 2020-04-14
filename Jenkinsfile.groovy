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
                             def image = env.AWS_URI + '/service-index'
                             def url = 'https://' + env.AWS_URI
                             def credentials = 'ecr:' + env.AWS_REGION + ':aws_credentials'
                             docker.withRegistry(url, credentials) {
                                docker.image(image).push('SNAPSHOT-${BUILD_NUMBER}')
                             }
                      }
                   }              
                }
            }

        }
}

