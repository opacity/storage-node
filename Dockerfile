FROM golang:1.15
ENV ADDR=0.0.0.0

RUN go version

RUN apt-get update
RUN apt-get install -y -q --no-install-recommends default-mysql-client
RUN apt-get install -y -q --no-install-recommends netcat
RUN apt-get update -y && apt-get install -y -q --no-install-recommends ffmpeg
RUN apt autoremove -y
RUN apt-get clean
RUN rm -rf /var/lib/apt/lists/*

RUN mkdir -p "$GOPATH/src/github.com/opacity/storage-node"
WORKDIR "$GOPATH/src/github.com/opacity/storage-node"

COPY . .

RUN go build ./...
