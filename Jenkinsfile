properties([disableConcurrentBuilds(), buildDiscarder(logRotator(artifactDaysToKeepStr: '5', artifactNumToKeepStr: '5', daysToKeepStr: '5', numToKeepStr: '5'))])

@Library('pipeline-library')
import dk.stiil.pipeline.Constants

podTemplate(yaml: '''
    apiVersion: v1
    kind: Pod
    spec:
      containers:
      - name: kaniko
        image: gcr.io/kaniko-project/executor:debug
        command:
        - sleep
        args: 
        - 99d
        volumeMounts:
        - name: kaniko-secret
          mountPath: /kaniko/.docker
      - name: golang
        image: golang:alpine
        command:
        - sleep
        args: 
        - 99d
      restartPolicy: Never
      volumes:
      - name: kaniko-secret
        secret:
          secretName: github-dockercred
          items:
          - key: .dockerconfigjson
            path: config.json
''') {
  node(POD_LABEL) {
    TreeMap scmData
    stage('checkout SCM') {  
      scmData = checkout scm
    }
    container('golang') {
      stage('UnitTests') {
        withEnv(['CGO_ENABLED=0', 'GOOS=linux', 'GOARCH=amd64']) {
          sh '''
            go test .
          '''
        }
      }
      stage('Build Application') {
        withEnv(['CGO_ENABLED=0', 'GOOS=linux', 'GOARCH=amd64']) {
          sh '''
            go build -ldflags="-w -s" .
          '''
        }
      }
      stage('Generate Dockerfile') {
        sh '''
          ./dockerfilegen.sh
        '''
      }
    }
    stage('Build Docker Image') {
      container('kaniko') {
        withEnv(["GIT_COMMIT=${scmData.GIT_COMMIT}"]) {
          sh '''
            /kaniko/executor --force --context `pwd` --log-format text --destination ghcr.io/simonstiil/kvdb:$BRANCH_NAME --label org.opencontainers.image.description=$GIT_COMMIT
          '''
        }
      }
    }
  }
}