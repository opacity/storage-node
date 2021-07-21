#!/bin/bash

cd /home/ubuntu/storage-node/
cp ../.env .
version=$(<.version)
go_env=$(<.go_env)
sed -i 's/VERSION=.*/VERSION='${version}'/' .env
sed -i 's/GO_ENV=.*/GO_ENV='${go_env}'/' .env

docker-compose up -d
