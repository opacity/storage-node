FROM prom/prometheus

USER root

RUN mkdir -p "$GOPATH/src/github.com/opacity/storage-node/prometheus"
WORKDIR "$GOPATH/src/github.com/opacity/storage-node/prometheus"

COPY . .

RUN chmod +x entrypoint.sh

# TODO:  We had to switch to user "root" to be able to run the commands,
# and ideally we'd run "RUN user nobody" to switch back to the user we started
# as.  However this causes an error during building.  Figure out how to solve the
# error.

WORKDIR "$GOPATH/src/github.com/opacity/storage-node"
