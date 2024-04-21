properties([disableConcurrentBuilds(), buildDiscarder(logRotator(artifactDaysToKeepStr: '5', artifactNumToKeepStr: '5', daysToKeepStr: '5', numToKeepStr: '5'))])

@Library('pipeline-library')
import dk.stiil.pipeline.Constants

String testpassword = generateAplhaNumericString( 16 )
String rootpassword = generateAplhaNumericString( 16 )

def template = '''
    apiVersion: v1
    kind: Pod
    spec:
      containers:
      - name: kaniko
        image: gcr.io/kaniko-project/executor:v1.22.0-debug
        command:
        - sleep
        args: 
        - 99d
        volumeMounts:
        - name: kaniko-secret
          mountPath: /kaniko/.docker
      - name: golang
        image: golang:1.22.2-alpine3.19
        command:
        - sleep
        args: 
        - 99d
      - name: mariadb
        image: mariadb:11.3.2-jammy
        env:
          - name: MARIADB_USER
            value: "kvdb"
          - name: MARIADB_PASSWORD
            value: "TEMPORARY_FAKE_PASSWORD"
          - name: MARIADB_DATABASE
            value: "kvdb-test"
          - name: MARIADB_ROOT_PASSWORD
            value: "TEMPORARY_FAKE_ROOT_PASSWORD"
        ports:
          - containerPort: 3306
      restartPolicy: Never
      nodeSelector: 
        kubernetes.io/arch: amd64
      volumes:
      - name: kaniko-secret
        secret:
          secretName: github-dockercred
          items:
          - key: .dockerconfigjson
            path: config.json
'''
template = template.replaceAll("TEMPORARY_FAKE_PASSWORD",testpassword).replaceAll("TEMPORARY_FAKE_ROOT_PASSWORD",rootpassword)

podTemplate(yaml: template) {
  node(POD_LABEL) {
    TreeMap scmData
    String gitCommitMessage
    stage('checkout SCM') {  
      scmData = checkout scm
      gitCommitMessage = sh(returnStdout: true, script: "git log --format=%B -n 1 ${scmData.GIT_COMMIT}").trim()
      gitMap = scmGetOrgRepo scmData.GIT_URL
      githubWebhookManager gitMap: gitMap, webhookTokenId: 'jenkins-webhook-repo-cleanup'
    }

    container('golang') {
      stage('UnitTests') {
        withEnv(['CGO_ENABLED=0', 'GOOS=linux', 'GOARCH=amd64', 'KVDB_DATABASETYPE=mysql', "KVDB_MYSQL_PASSWORD=${testpassword}"]) {
          sh '''
            go test . -v 
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
    if ( !gitCommitMessage.startsWith("renovate/") || ! gitCommitMessage.startsWith("WIP") ) {
      stage('Build Docker Image') {
        container('kaniko') {
          def properties = readProperties file: 'package.env'
          withEnv(["GIT_COMMIT=${scmData.GIT_COMMIT}", "PACKAGE_NAME=${properties.PACKAGE_NAME}", "PACKAGE_DESTINATION=${properties.PACKAGE_DESTINATION}", "PACKAGE_CONTAINER_SOURCE=${properties.PACKAGE_CONTAINER_SOURCE}", "GIT_BRANCH=${BRANCH_NAME}"]) {
            if (isMainBranch()){
              sh '''
                /kaniko/executor --force --context `pwd` --log-format text --destination $PACKAGE_DESTINATION/$PACKAGE_NAME:$BRANCH_NAME --destination $PACKAGE_DESTINATION/$PACKAGE_NAME:latest --label org.opencontainers.image.description="Build based on $PACKAGE_CONTAINER_SOURCE/commit/$GIT_COMMIT" --label org.opencontainers.image.revision=$GIT_COMMIT --label org.opencontainers.image.version=$GIT_BRANCH
              '''
            } else {
              sh '''
                /kaniko/executor --force --context `pwd` --log-format text --destination $PACKAGE_DESTINATION/$PACKAGE_NAME:$BRANCH_NAME --label org.opencontainers.image.description="Build based on $PACKAGE_CONTAINER_SOURCE/commit/$GIT_COMMIT" --label org.opencontainers.image.revision=$GIT_COMMIT --label org.opencontainers.image.version=$GIT_BRANCH

              '''
            }
          }
        }
      }
    }
  }
}
