pipeline {
    agent any
    environment {
        REPO_NAME = 'prime-wms-erp-core'      
        IMAGE_NAME = 'wms-erp-core'
        PORT = '9115:9115' 
        CONTAINER_NAME = 'wms-erp-core-container'
        TARGET_BRANCH = 'Kernel-Develop' 
        REMOTE_USER = 'ec2-user'
        REMOTE_HOST = '47.128.166.228'
        SSH_KEY_PATH = '/home/ec2-user/key/kernal-uat.pem'
    }
    stages {
        stage('Check SSH Key Access and User') {
            steps {
                script {
                    echo "Check current Jenkins user"
                    sh 'whoami'

                    echo "Check SSH key file permission and existence"
                    sh "ls -l ${SSH_KEY_PATH} || echo 'SSH key file not found or no permission'"

                    echo "Test reading SSH key file content (first line only)"
                    sh "head -1 ${SSH_KEY_PATH} || echo 'Cannot read SSH key file'"

                    echo "Test ssh command dry run (no login)"
                    sh "ssh -o BatchMode=yes -o ConnectTimeout=5 -i ${SSH_KEY_PATH} ${REMOTE_USER}@${REMOTE_HOST} 'echo SSH connection test successful' || echo 'SSH connection test failed'"
                }
            }
        }
        stage('SSH to Remote Server') {
            steps {
                script {
                    echo "Testing SSH connection to ${REMOTE_USER}@${REMOTE_HOST}"
                    sh "ssh -o StrictHostKeyChecking=no -i ${SSH_KEY_PATH} ${REMOTE_USER}@${REMOTE_HOST} 'pwd'"
                }
            }
        }
        stage('Clone Repository') {
            steps {
                script {
                    echo "Cloning or updating repository branch: ${TARGET_BRANCH} on remote server..."
                    withCredentials([string(credentialsId: 'GITTOKEN', variable: 'GIT_TOKEN')]) {
                        sh """ssh -o StrictHostKeyChecking=no -i ${SSH_KEY_PATH} ${REMOTE_USER}@${REMOTE_HOST} \\
                        'git clone -b ${TARGET_BRANCH} https://${GIT_TOKEN}@github.com/prime-solution-dev/${REPO_NAME} || \\
                        (cd ${REPO_NAME} && git fetch && git checkout ${TARGET_BRANCH} && git pull origin ${TARGET_BRANCH})'"""
                    }
                    echo 'Repository cloned/updated successfully on remote!'
                }
            }
        }
        stage('Check Workspace') {
            steps {
                script {
                    sh "ssh -i ${SSH_KEY_PATH} ${REMOTE_USER}@${REMOTE_HOST} 'pwd && ls -la ${REPO_NAME}'"
                }
            }
        }
        stage('Backup Current Image') {
            steps {
                script {
                    def timestamp = new Date().format("yyyyMMdd-HHmmss", TimeZone.getTimeZone('UTC'))
                    def backupImageName = "${IMAGE_NAME}-backup:${timestamp}"
                    def exportPath = "/wms/backup_wms/${IMAGE_NAME}-backup-${timestamp}.tar"

                    echo 'Backing up current image on remote server...'
                    sh """ssh -i ${SSH_KEY_PATH} ${REMOTE_USER}@${REMOTE_HOST} \\
                    "sourceImage=\\\$(docker images --filter=reference='${IMAGE_NAME}:latest' -q); \\
                    if [ -n \\\"\\\$sourceImage\\\" ]; then \\
                        docker tag ${IMAGE_NAME}:latest ${backupImageName}; \\
                        docker save -o ${exportPath} ${backupImageName}; \\
                        echo 'Backup created and exported to ${exportPath}'; \\
                    else \\
                        echo 'No source image found, skipping backup.'; \\
                    fi" """
                    echo 'Cleaning up old backups on remote server...'
                    sh """ssh -i ${SSH_KEY_PATH} ${REMOTE_USER}@${REMOTE_HOST} \\
                    "docker images --filter=reference='${IMAGE_NAME}-backup:*' --format '{{.Repository}}:{{.Tag}}' | sort | head -n -5 | xargs -r docker rmi -f" """
                }
            }
        }
        stage('Remove Old Docker Image') {
            steps {
                script {
                    echo 'Removing old Docker image on remote server...'
                    sh """ssh -i ${SSH_KEY_PATH} ${REMOTE_USER}@${REMOTE_HOST} docker rmi -f ${IMAGE_NAME}:latest || true"""
                }
            }
        }
stage('Update .env with DB_PASSWORD') {
    steps {
        script {
            echo 'Updating .env file with DB_PASSWORD on remote...'
            withCredentials([string(credentialsId: 'Jenkinskernaluatserver', variable: 'DB_PASSWORD')]) {
                sh """
                ssh -i ${SSH_KEY_PATH} ${REMOTE_USER}@${REMOTE_HOST} \\
                "cd ${REPO_NAME} && \\
                 if [ -f cmd/.env ]; then \\
                     sed -i 's|\\\${DB_PASSWORD}|${DB_PASSWORD}|g' cmd/.env && \\
                     echo '.env updated successfully!' && \\
                     cat cmd/.env; \\
                 else \\
                     echo '.env file not found!'; \\
                 fi"
                """
            }
        }
    }
}
stage('Build Docker Image') {
            steps {
                script {
                    echo 'Building Docker image on remote server...'
                    sh """ssh -i ${SSH_KEY_PATH} ${REMOTE_USER}@${REMOTE_HOST} \\
                    'cd ${REPO_NAME} && docker build --no-cache -t  ${IMAGE_NAME}:latest .'"""
                }
            }
        }
        stage('Stop and Remove Old Container') {
            steps {
                script {
                    echo 'Stopping and removing old container on remote server...'
                    sh """ssh -i ${SSH_KEY_PATH} ${REMOTE_USER}@${REMOTE_HOST} \\
                    "docker ps -aq --filter name=${CONTAINER_NAME} | xargs -r docker stop && docker ps -aq --filter name=${CONTAINER_NAME} | xargs -r docker rm" """
                }
            }
        }
        stage('Deploy Container') {
            steps {
                script {
                    echo 'Deploying container on remote server...'
                    sh """ssh -i ${SSH_KEY_PATH} ${REMOTE_USER}@${REMOTE_HOST} \\
                    "docker run -d -p ${PORT} --cpus=1.0 --name ${CONTAINER_NAME} ${IMAGE_NAME}:latest" """
                }
            }
        }
    }
    post {
        success {
            echo 'Pipeline completed successfully!'
        }
        failure {
            script {
                echo 'Pipeline failed. Attempting rollback on remote server...'

                sh """ssh -i ${SSH_KEY_PATH} ${REMOTE_USER}@${REMOTE_HOST} \\
                "docker ps -aq --filter name=${CONTAINER_NAME} | xargs -r docker stop && docker ps -aq --filter name=${CONTAINER_NAME} | xargs -r docker rm" """

                def latestBackup = sh(script: "ssh -i ${SSH_KEY_PATH} ${REMOTE_USER}@${REMOTE_HOST} docker images --filter=reference='${IMAGE_NAME}-backup:*' --format '{{.Repository}}:{{.Tag}}' | sort | tail -n 1", returnStdout: true).trim()

                if (latestBackup) {
                    echo "Found backup image: ${latestBackup}, deploying..."
                    sh """ssh -i ${SSH_KEY_PATH} ${REMOTE_USER}@${REMOTE_HOST} \\
                    "docker run -d -p ${PORT} --cpus=1.0 --name ${CONTAINER_NAME} ${latestBackup}" """
                    echo "Rollback completed using backup image: ${latestBackup}"
                } else {
                    echo "No backup image found. Rollback cannot be completed."
                }
            }
        }
    }
}
