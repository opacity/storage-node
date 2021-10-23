FROM influxdb:1.7.8

COPY influxdb.sh /docker-entrypoint-initdb.d/influxdb.sh
COPY influxdb.conf /etc/influxdb/influxdb.conf:ro
COPY prometheus.iql /docker-entrypoint-initdb.d/prometheus.iql:ro