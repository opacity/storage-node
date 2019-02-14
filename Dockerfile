FROM golang:1.11
ENV ADDR=0.0.0.0

RUN go version

# Install db client (assumes mysql)
RUN apt-get update
RUN apt-get install -y -q mysql-client
RUN apt-get install -y -q netcat

RUN mkdir -p $GOPATH/src/github.com/opacity/storage-node
WORKDIR $GOPATH/src/github.com/opacity/storage-node

RUN go get -u -v github.com/kardianos/govendor

COPY . .

RUN govendor sync
RUN go build
