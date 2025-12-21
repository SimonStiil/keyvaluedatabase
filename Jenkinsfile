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
      - name: buildkit
        image: moby/buildkit:v0.26.3-rootless # renovate
        command:
        - /bin/sh
        tty: true
        volumeMounts:
        - name: docker-secret
          mountPath: /home/user/.docker
        - name: certs
          mountPath: /certs/client
      - name: golang
        image: golang:1.25.5-alpine # renovate
        command:
        - sleep
        args: 
        - 99d
        env:
        - name: HOST_NAME
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: spec.nodeName
        volumeMounts:
        - name: "golang-cache"
          mountPath: "/root/.cache/"
        - name: "golang-prgs"
          mountPath: "/go/pkg/"
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
      - name: postgres
        image: postgres:18.1-alpine
        env:
          - name: POSTGRES_USER
            value: "kvdb"
          - name: POSTGRES_PASSWORD
            value: "TEMPORARY_FAKE_PASSWORD"
          - name: POSTGRES_DB
            value: "kvdb-test"
        ports:
          - containerPort: 5432
      - name: redis
        image: redis:8.2.3
        command:
          - redis-server
          - --requirepass
          - TEMPORARY_FAKE_PASSWORD
        ports:
          - containerPort: 6379
      restartPolicy: Never
      volumes:
      - name: docker-secret
        secret:
          secretName: github-dockercred
          items:
          - key: .dockerconfigjson
            path: config.json
      - name: certs
        secret:
          secretName: buildkit-client-certs
      - name: "golang-cache"
        persistentVolumeClaim:
          claimName: "golang-cache"
      - name: "golang-prgs"
        persistentVolumeClaim:
          claimName: "golang-prgs"
'''
template = template.replaceAll("TEMPORARY_FAKE_PASSWORD",testpassword).replaceAll("TEMPORARY_FAKE_ROOT_PASSWORD",rootpassword)

podTemplate(yaml: template) {
  node(POD_LABEL) {
    TreeMap scmData
    String gitCommitMessage
    Map properties
    stage('checkout SCM') {  
      scmData = checkout scm
      gitCommitMessage = sh(returnStdout: true, script: "git log --format=%B -n 1 ${scmData.GIT_COMMIT}").trim()
//      gitMap = scmGetOrgRepo scmData.GIT_URL
//      githubWebhookManager gitMap: gitMap, webhookTokenId: 'jenkins-webhook-repo-cleanup'
      properties = readProperties file: 'package.env'
    }

    container('golang') {
      stage('Install tools') {
        currentBuild.description = sh(returnStdout: true, script: 'echo $HOST_NAME').trim()
        sh '''
            apk --update add openssl git
            df -h /root/.cache /go/pkg
            go install github.com/jstemmer/go-junit-report/v2@v2.1.0
            ./generate-test-cert.sh
        '''
      }
      stage('UnitTests') {
        currentBuild.description = sh(returnStdout: true, script: 'echo $HOST_NAME').trim()
        try{
          withEnv(['CGO_ENABLED=0', 'KVDB_DATABASETYPE=postgres', "KVDB_MYSQL_PASSWORD=${testpassword}", "KVDB_POSTGRES_PASSWORD=${testpassword}", "KVDB_REDIS_PASSWORD=${testpassword}"]) {
            sh '''
              go test . -v -tags="unit integration" -covermode=atomic -coverprofile=coverage.out 2>&1 > tests.out
              go-junit-report -in tests.out -iocopy -out report.xml -set-exit-code
              go tool cover -func coverage.out
            '''
            
          }
        } catch (Exception e) {
          def fileContent = readFile('tests.out')
          echo fileContent
          junit 'report.xml'
          archiveArtifacts artifacts: 'report.xml', fingerprint: true
          echo 'Exception occurred: ' + e.toString()
          throw e
        }
        junit 'report.xml'
        archiveArtifacts artifacts: 'report.xml', fingerprint: true
      }
      stage('Build Application AMD64') {
        withEnv(['CGO_ENABLED=0', 'GOOS=linux', 'GOARCH=amd64', "PACKAGE_CONTAINER_APPLICATION=${properties.PACKAGE_CONTAINER_APPLICATION}"]) {
          sh '''
            go build -buildvcs=false -ldflags="-w -s" -o $PACKAGE_CONTAINER_APPLICATION-amd64 .
          '''
        }
      }
      stage('Build Application ARM64') {
        withEnv(['CGO_ENABLED=0', 'GOOS=linux', 'GOARCH=arm64', "PACKAGE_CONTAINER_APPLICATION=${properties.PACKAGE_CONTAINER_APPLICATION}"]) {
          sh '''
            go build -buildvcs=false -ldflags="-w -s" -o $PACKAGE_CONTAINER_APPLICATION-arm64 .
          '''
        }
      }
    }

    String name
    if (isMainBranch()){
      pushName = "\"name=${properties.PACKAGE_DESTINATION}/${properties.PACKAGE_NAME}:$BRANCH_NAME,${properties.PACKAGE_DESTINATION}/${properties.PACKAGE_NAME}:latest\""
    } else {
      pushName = "name=${properties.PACKAGE_DESTINATION}/${properties.PACKAGE_NAME}:$BRANCH_NAME"
    }
    if ( !gitCommitMessage.startsWith("renovate/") || ! gitCommitMessage.startsWith("WIP") ) {
      container('buildkit') {
        stage('Build Docker Image') {
          withEnv(["GIT_COMMIT=${scmData.GIT_COMMIT}", "PUSH_NAME=${pushName}", "PACKAGE_NAME=${properties.PACKAGE_NAME}", "PACKAGE_CONTAINER_PLATFORMS=${properties.PACKAGE_CONTAINER_PLATFORMS}", "PACKAGE_DESTINATION=${properties.PACKAGE_DESTINATION}", "PACKAGE_CONTAINER_SOURCE=${properties.PACKAGE_CONTAINER_SOURCE}", "GIT_BRANCH=${BRANCH_NAME}"]) {
            sh '''
              buildctl --addr 'tcp://buildkitd:1234'\
              --tlscacert /certs/client/ca.crt \
              --tlscert /certs/client/tls.crt \
              --tlskey /certs/client/tls.key \
              build \
              --frontend dockerfile.v0 \
              --opt filename=Dockerfile --opt platform=$PACKAGE_CONTAINER_PLATFORMS \
              --local context=$(pwd) --local dockerfile=$(pwd) \
              --import-cache $PACKAGE_DESTINATION/$PACKAGE_NAME:buildcache \
              --export-cache $PACKAGE_DESTINATION/$PACKAGE_NAME:buildcache \
              --output type=image,$PUSH_NAME,push=true,annotation.org.opencontainers.image.description="Build based on $PACKAGE_CONTAINER_SOURCE/commit/$GIT_COMMIT",annotation.org.opencontainers.image.revision=$GIT_COMMIT,annotation.org.opencontainers.image.version=$GIT_BRANCH,annotation.org.opencontainers.image.source=$PACKAGE_CONTAINER_SOURCE,annotation.org.opencontainers.image.licenses=GPL-2.0-only
              '''
          }
        }
      }
      if (env.CHANGE_ID) {
        if (pullRequest.createdBy.equals("renovate[bot]")){
          if (pullRequest.mergeable) {
            stage('Approve and Merge PR') {
              pullRequest.merge(commitTitle: pullRequest.title, commitMessage: pullRequest.body, mergeMethod: 'squash')
            }
          }
        } else {
          echo "'PR Created by \""+ pullRequest.createdBy + "\""
        }
      }
    }
  }
}
