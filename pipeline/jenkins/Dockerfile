# Jenkins-контролер, який одночасно є агентом збірки kbot (agent any):
# jenkins/jenkins:lts + Go, make, docker CLI/buildx, плагіни та JCasC-конфіг.
#
#   docker build -t kbot-jenkins pipeline/jenkins
#   docker run -d --name kbot-jenkins -p 8081:8080 \
#     -v kbot-jenkins-home:/var/jenkins_home \
#     -v /var/run/docker.sock:/var/run/docker.sock --group-add 0 \
#     kbot-jenkins
FROM jenkins/jenkins:lts-jdk21

ARG GO_VERSION=1.26.8
ARG TARGETARCH=amd64

USER root

RUN apt-get update \
 && apt-get install -y --no-install-recommends make ca-certificates curl gnupg \
 && install -m 0755 -d /etc/apt/keyrings \
 && curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc \
 && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
      > /etc/apt/sources.list.d/docker.list \
 && apt-get update \
 && apt-get install -y --no-install-recommends docker-ce-cli docker-buildx-plugin \
 && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz" | tar -C /usr/local -xz
ENV PATH=/usr/local/go/bin:$PATH

# Плагіни та конфігурація (setup wizard вимкнено, все описано в casc.yaml)
COPY plugins.txt /usr/share/jenkins/ref/plugins.txt
RUN jenkins-plugin-cli --plugin-file /usr/share/jenkins/ref/plugins.txt
COPY casc.yaml /usr/local/etc/jenkins/casc.yaml
ENV CASC_JENKINS_CONFIG=/usr/local/etc/jenkins/casc.yaml \
    JAVA_OPTS="-Djenkins.install.runSetupWizard=false"

USER jenkins
