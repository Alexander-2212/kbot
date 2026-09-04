// Jenkins Declarative Pipeline: параметризована мультиплатформенна збірка kbot.
//
// Агент - хост/контейнер, на якому розгорнуто Jenkins (agent any); на ньому мають бути
// go, make, git і (для стадій Image/Push) docker CLI з доступом до Docker daemon.
// Підключення: New Item -> Pipeline -> "Pipeline script from SCM" ->
//   Repository: https://github.com/Alexander-2212/kbot.git, Branch: */develop,
//   Script Path: pipeline/jenkins.groovy
//
// Розробник обирає параметри збірки у формі "Build with Parameters" або
// використовує значення за замовчуванням (linux/amd64, з лінтером і тестами).

pipeline {
    agent any

    parameters {
        choice(
            name: 'OS',
            choices: ['linux', 'darwin', 'windows'],
            description: 'Цільова операційна система (GOOS)'
        )
        choice(
            name: 'ARCH',
            choices: ['amd64', 'arm64'],
            description: 'Цільова архітектура (GOARCH)'
        )
        booleanParam(
            name: 'SKIP_TESTS',
            defaultValue: false,
            description: 'Пропустити тести (make test)'
        )
        booleanParam(
            name: 'SKIP_LINT',
            defaultValue: false,
            description: 'Пропустити лінтер (gofmt, go vet)'
        )
        booleanParam(
            name: 'BUILD_IMAGE',
            defaultValue: false,
            description: 'Зібрати контейнерний образ (docker buildx); лише для OS=linux'
        )
        booleanParam(
            name: 'PUSH_IMAGE',
            defaultValue: false,
            description: 'Запушити образ у REGISTRY (потрібні credentials з id ghcr-credentials)'
        )
        string(
            name: 'RELEASE',
            defaultValue: 'v1.0.0',
            description: 'Префікс версії; VERSION = <RELEASE>-<short SHA>'
        )
    }

    options {
        buildDiscarder(logRotator(numToKeepStr: '20'))
        disableConcurrentBuilds()
        timeout(time: 30, unit: 'MINUTES')
        skipDefaultCheckout()
    }

    // Makefile читає ці змінні з оточення (оператор ?=), тож stages викликають make без зайвих аргументів.
    environment {
        APP         = 'kbot'
        REGISTRY    = 'ghcr.io'
        REPOSITORY  = 'alexander-2212/kbot'
        TARGETOS    = "${params.OS}"
        TARGETARCH  = "${params.ARCH}"
        RELEASE     = "${params.RELEASE}"
        CGO_ENABLED = '0'
        // кеші Go всередині workspace, щоб не залежати від $HOME агента
        GOCACHE     = "${WORKSPACE}/.cache/go-build"
        GOMODCACHE  = "${WORKSPACE}/.cache/go-mod"
        GOTOOLCHAIN = 'auto'
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
                script {
                    env.COMMIT  = sh(script: 'git rev-parse --short=7 HEAD', returnStdout: true).trim()
                    env.VERSION = "${env.RELEASE}-${env.COMMIT}"
                    env.BIN     = "${env.APP}-${env.TARGETOS}-${env.TARGETARCH}" + (params.OS == 'windows' ? '.exe' : '')
                    env.IMAGE   = sh(script: 'make -s image-name', returnStdout: true).trim()
                    currentBuild.displayName = "#${env.BUILD_NUMBER} ${env.VERSION} ${env.TARGETOS}/${env.TARGETARCH}"
                    currentBuild.description = "bin: ${env.BIN}" + (params.BUILD_IMAGE ? "<br/>image: ${env.IMAGE}" : '')
                }
                sh '''
                    go version
                    make help
                '''
            }
        }

        stage('Lint') {
            when { expression { !params.SKIP_LINT } }
            steps {
                sh '''
                    unformatted="$(gofmt -l .)"
                    if [ -n "$unformatted" ]; then
                        echo "gofmt: файли не відформатовано:"; echo "$unformatted"; exit 1
                    fi
                    make vet
                '''
            }
        }

        stage('Test') {
            when { expression { !params.SKIP_TESTS } }
            steps {
                sh 'make test'
            }
        }

        stage('Build') {
            steps {
                sh '''
                    make build BIN="$BIN"
                    ls -l "$BIN"
                '''
                archiveArtifacts artifacts: "${env.BIN}", fingerprint: true
            }
        }

        stage('Image') {
            when {
                allOf {
                    expression { params.BUILD_IMAGE }
                    expression { params.OS == 'linux' }
                }
            }
            steps {
                sh '''
                    docker buildx version
                    make image
                    docker image inspect "$IMAGE" --format 'built {{.Id}} {{.Os}}/{{.Architecture}} size={{.Size}}'
                '''
            }
        }

        stage('Push') {
            when {
                allOf {
                    expression { params.BUILD_IMAGE && params.PUSH_IMAGE }
                    expression { params.OS == 'linux' }
                }
            }
            steps {
                withCredentials([usernamePassword(credentialsId: 'ghcr-credentials',
                                                  usernameVariable: 'REG_USER',
                                                  passwordVariable: 'REG_TOKEN')]) {
                    sh '''
                        echo "$REG_TOKEN" | docker login "$REGISTRY" -u "$REG_USER" --password-stdin
                        make push
                    '''
                }
            }
            post {
                always {
                    sh 'docker logout "$REGISTRY" >/dev/null 2>&1 || true'
                }
            }
        }
    }

    post {
        success {
            echo "OK: ${env.VERSION} ${env.TARGETOS}/${env.TARGETARCH} -> ${env.BIN}" +
                 (params.BUILD_IMAGE && params.OS == 'linux' ? ", образ ${env.IMAGE}" : '')
        }
        failure {
            echo "FAILED: ${env.VERSION} ${env.TARGETOS}/${env.TARGETARCH} (stage: ${env.STAGE_NAME})"
        }
    }
}
