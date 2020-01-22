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
                }
            }

        }
}

