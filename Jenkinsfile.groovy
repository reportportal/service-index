#!groovy

node {

       load "$JENKINS_HOME/jobvars.env"

       dir('src/github.com/reportportal/service-index') {

           stage('Checkout'){
                checkout scm
                sh 'git checkout master'
                sh 'git pull'
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
                   sh "docker-compose -p reportportal5 -f $COMPOSE_FILE_RP_5 up -d --force-recreate index"
                }
            }

        }
}

