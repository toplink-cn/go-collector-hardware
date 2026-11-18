pipeline {
    agent any
    tools { go '1.20.3' }
    stages {
        stage('Verify') {
            steps { sh 'go test -race ./...' }
        }
        stage('Package') {
            steps { sh 'scripts/package.sh' }
        }
        stage('Artifacts') {
            steps { archiveArtifacts artifacts: 'dist/*.tar.gz,dist/*.zip,dist/SHA256SUMS', fingerprint: true }
        }
    }
}
