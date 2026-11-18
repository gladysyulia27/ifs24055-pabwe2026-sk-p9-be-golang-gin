pipeline {
  agent any
  stages {
    stage('Checkout') { steps { checkout scm } }
    stage('Dependencies') { steps { sh 'go version && go mod download && go mod verify' } }
    stage('Test') { steps { sh 'go test -v -coverprofile=coverage.out ./internal/...' } }
    stage('Build') { steps { sh 'CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server' } }
  }
  post { always { archiveArtifacts artifacts: 'coverage.out', allowEmptyArchive: true } }
}
