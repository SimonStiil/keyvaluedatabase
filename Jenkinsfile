properties([disableConcurrentBuilds(), buildDiscarder(logRotator(artifactDaysToKeepStr: '5', artifactNumToKeepStr: '5', daysToKeepStr: '5', numToKeepStr: '5'))])

@Library('pipeline-library@devops-github')
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
      gitMap = scmGetOrgRepo scmData.GIT_URL
      if (gitMap.host == "github.com") {
        def url = env.JENKINS_URL + "generic-webhook-trigger/invoke?token="
        def events = ["delete", "push"]
      //https://docs.github.com/en/rest/repos/webhooks?apiVersion=2022-11-28
        withCredentials([usernamePassword(credentialsId: "github-login-secret",
                                          usernameVariable : 'GITHUB_USERNAME',
                                          passwordVariable: 'GITHUB_PERSONAL' )]) {
          def response = httpRequest customHeaders: [[name: 'Accept', value: 'application/vnd.github+json'],
                                                     [name: 'Authorization', value: 'Bearer '+GITHUB_PERSONAL],
                                                     [name: 'X-GitHub-Api-Version', value: '2022-11-28']],
                  url: "https://api.github.com/repos/${gitMap.fullName}/hooks"
          def jsonResponse = readJSON text: response.content
          def configFound = false
          def configCorrect = false
          for (item in jsonResponse) {
            if (item.config.url.contains(url)) {
              configFound = true
              def found = true
              for (event in item.events) {
                def foundEvent = false
                for (matchEvent in events) {
                  if (event == matchEvent) {
                    foundEvent = true
                  }
                }
                if (!foundEvent){
                  found = false
                }
              }
              if (found)
              {
                echo "WebHook: Already configured"
                configCorrect = true
              } else {
                echo "WebHook: Configured but not matching events"
              }
            }
          }
          if (!configFound) {
            echo "WebHook: No config found"
          }
        }
      } else {
        echo "WebHook: Repository host ${gitMap.host} is not github.com"
      }
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
        def properties = readProperties file: 'package.env'
        withEnv(["GIT_COMMIT=${scmData.GIT_COMMIT}", "PACKAGE_NAME=${properties.PACKAGE_NAME}", "GIT_BRANCH=${BRANCH_NAME}"]) {
          if (isMainBranch()){
            sh '''
              /kaniko/executor --force --context `pwd` --log-format text --destination ghcr.io/simonstiil/$PACKAGE_NAME:$BRANCH_NAME --destination ghcr.io/simonstiil/$PACKAGE_NAME:latest --label org.opencontainers.image.description="Build based on https://github.com/SimonStiil/keyvaluedatabase/commit/$GIT_COMMIT" --label org.opencontainers.image.revision=$GIT_COMMIT --label org.opencontainers.image.version=$GIT_BRANCH
            '''
          } else {
            sh '''
              /kaniko/executor --force --context `pwd` --log-format text --destination ghcr.io/simonstiil/$PACKAGE_NAME:$BRANCH_NAME --label org.opencontainers.image.description="Build based on https://github.com/SimonStiil/keyvaluedatabase/commit/$GIT_COMMIT" --label org.opencontainers.image.revision=$GIT_COMMIT --label org.opencontainers.image.version=$GIT_BRANCH
            '''
          }
        }
        
      }
    }
  }
}