FROM prom/prometheus
ARG ADMIN_USER
ARG ADMIN_PASSWORD

USER root

RUN mkdir -p "$GOPATH/src/github.com/opacity/storage-node/prometheus"
WORKDIR "$GOPATH/src/github.com/opacity/storage-node/prometheus"

COPY . .

RUN sed -i -r "s/@@ADMIN_USER@@/$ADMIN_USER/" prometheus.yml
RUN sed -i -r "s/@@ADMIN_PASSWORD@@/$ADMIN_PASSWORD/" prometheus.yml

RUN cp prometheus.yml /etc/prometheus/prometheus.yml
VOLUME /etc/prometheus

WORKDIR "$GOPATH/src/github.com/opacity/storage-node"