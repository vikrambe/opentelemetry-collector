def label = "opentelemetry-container-${UUID.randomUUID().toString()}"


podTemplate(name: "opentelemetry-container", label: label, volumes: [hostPathVolume(hostPath: '/var/run/dind/docker.sock', mountPath: '/var/run/docker.sock')], containers:[
  containerTemplate(name: 'docker', image: 'golang:1.13', ttyEnabled: true, command: 'cat', args: '' ),
  containerTemplate(name: 'cpd', image: 'docker.intuit.com/oicp/standard/cpd:0.4', ttyEnabled: true, command: 'cat', args: '', alwaysPullImage: true)]) {

  node(label) {
    ansiColor('xterm') {
    stage('Checkout') {
      checkout scm
    }

    stage('Docker Build') {
          container('docker') {
            sh 'make otelcol'
            sh 'apt-get update ; apt-get install docker.io -y ; bash'
            sh 'make docker-otelcol'
            sh 'docker tag otelcol:latest docker.artifactory.a.intuit.com/cloud/logging/transaction-tracing/service/analytics/opentelemetry/service/otelsvc:0.2.7-extended'
            }
          if (env.CHANGE_ID) {
            currentBuild.result = 'SUCCESS'
            return    
          }
    }
      
       stage('CPD Certification') {
        withCredentials([usernamePassword(credentialsId: "twistlock-cpd-scan", passwordVariable: 'SCAN_PASSWORD', usernameVariable: 'SCAN_USER'), usernamePassword(credentialsId: "artifactory-jaeger", passwordVariable: 'DOCKER_PASSWORD', usernameVariable: 'DOCKER_USERNAME')]) {                  
            withEnv(['SCAN_SERVER=https://artifactscan.a.intuit.com:8083']) {
                container('cpd') {
                    sh "/cpd --buildargs DOCKER_IMAGE_NAME=docker.artifactory.a.intuit.com/cloud/logging/transaction-tracing/service/analytics/opentelemetry/service/otelsvc:0.2.7-extended -publish"
                 }
            }
        }
    }   

    }
    
    }

  }
