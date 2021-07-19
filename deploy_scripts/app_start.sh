#!/bin/bash

cd /home/ubuntu/storage-node/
cp ../.env .
version=$(<../.version)
sed -i 's/VERSION=.*/VERSION="'${version}'"/' .env

docker-compose up -d
