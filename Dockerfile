FROM golang:1.17
ENV ADDR=0.0.0.0

RUN go version

RUN apt-get update && apt-get install -y -q --no-install-recommends default-mysql-client netcat unar
RUN apt autoremove -y
RUN apt-get clean
RUN rm -rf /var/lib/apt/lists/*

RUN curl -O -L https://aws-codedeploy-deployments-dev.s3.us-east-2.amazonaws.com/ffmpeg-4.2.2-opacity-amd64.tar.xz
RUN unar ffmpeg-4.2.2-opacity-amd64.tar.xz && cd ffmpeg-4.2.2-opacity-amd64 && chmod a+x * && mv ff* /usr/local/bin/

RUN mkdir -p "$GOPATH/src/github.com/opacity/storage-node"
WORKDIR "$GOPATH/src/github.com/opacity/storage-node"

COPY . .

RUN go build ./...
