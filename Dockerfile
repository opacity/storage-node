FROM golang:1.11
ENV ADDR=0.0.0.0
ENV GO111MODULE=on

RUN go version

# Install db client (assumes mysql)
RUN apt-get update
RUN apt-get install -y -q --no-install-recommends mysql-client
RUN apt-get install -y -q --no-install-recommends netcat
RUN apt autoremove -y
RUN apt-get clean
RUN rm -rf /var/lib/apt/lists/*

RUN mkdir -p "$GOPATH/src/github.com/opacity/storage-node"
WORKDIR "$GOPATH/src/github.com/opacity/storage-node"

COPY . .

RUN go build ./...
