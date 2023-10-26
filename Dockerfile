FROM golang:1.20 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o atlantis-drift-detector .

FROM alpine:3.18.4 as alpine 

USER root

COPY --from=builder /app/atlantis-drift-detector .
COPY --from=builder /app/static /static

ENV TERRAFORM_VERSION="1.5.6"
ENV TERRAGRUNT_VERSION="0.45.0"
ENV YQ_VERSION="4.30.8"
ENV KUBECTL_VERSION="1.26.1"
ENV HELM_VERSION="3.11.2"
ENV HELM_GCS_PLUGIN_VERSION="0.4.0"
ENV YTT_VERSION="0.45.1"
ENV ARCH="amd64"

RUN mkdir -p /usr/local/sbin && mkdir -p /home/atlantis/.config/helm && \
    cd /usr/local/sbin && \
    apk --update --upgrade add gcc musl-dev jpeg-dev zlib-dev libffi-dev cairo-dev pango-dev gdk-pixbuf-dev python3 py3-pip python3-dev tar gzip unzip curl git && \
    wget https://github.com/gruntwork-io/terragrunt/releases/download/v$TERRAGRUNT_VERSION/terragrunt_linux_amd64 && \
    mv terragrunt_linux_amd64 terragrunt && \
    chmod +x terragrunt && \
    wget https://github.com/mikefarah/yq/releases/download/v$YQ_VERSION/yq_linux_amd64 && \
    mv yq_linux_amd64 yq && \
    chmod +x yq && \
    wget https://get.helm.sh/helm-v$HELM_VERSION-linux-amd64.tar.gz && \
    tar -zxvf helm-v$HELM_VERSION-linux-amd64.tar.gz && \
    mv linux-amd64/helm helm && \
    rm -rf helm-v$HELM_VERSION-linux-amd64.tar.gz && \
    rm -rf linux-amd64 && \
    wget https://github.com/carvel-dev/ytt/releases/download/v$YTT_VERSION/ytt-linux-amd64 && \
    wget https://github.com/carvel-dev/ytt/releases/download/v$YTT_VERSION/checksums.txt && \
    cat checksums.txt | grep ytt-linux-amd64 | sha256sum -c - && \
    mv ytt-linux-amd64 ytt && \
    chmod +x ytt && \
    rm checksums.txt && \
    wget https://storage.googleapis.com/kubernetes-release/release/v$KUBECTL_VERSION/bin/linux/amd64/kubectl && \
    chmod +x kubectl && \
    curl -LOs "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_${ARCH}.zip" && \
    unzip "terraform_${TERRAFORM_VERSION}_linux_${ARCH}.zip" && \
    rm -rf "terraform_${TERRAFORM_VERSION}_linux_${ARCH}.zip" && \
    chmod +x terraform && \
    pip3 install python-openstackclient && \
    helm plugin install https://github.com/hayorov/helm-gcs.git --version $HELM_GCS_PLUGIN_VERSION


EXPOSE 8080

CMD ["./atlantis-drift-detector"]
