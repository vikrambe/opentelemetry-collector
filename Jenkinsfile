def label = "open-telemetry-${UUID.randomUUID().toString()}"

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
            }
          if (env.CHANGE_ID) {
            currentBuild.result = 'SUCCESS'
            return    
          }
    }

    }
    
    }

  }
