#!/bin/sh

sed -i -r "s/@@ADMIN_USER@@/$1/" prometheus/prometheus.yml
sed -i -r "s/@@ADMIN_PASSWORD@@/$2/" prometheus/prometheus.yml

/bin/prometheus --config.file=prometheus/prometheus.yml --web.external-url=http://localhost:12321/prometheus --log.level=debug --storage.tsdb.path=/prometheus --storage.tsdb.retention.time=1h