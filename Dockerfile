FROM golang:1.11
ENV ADDR=0.0.0.0

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

RUN go get -u -v github.com/kardianos/govendor
RUN go get github.com/codegangsta/gin
RUN gin -h

COPY . .

RUN govendor sync
RUN go build