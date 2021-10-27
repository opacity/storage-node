#!/bin/bash

cd /home/ubuntu/storage-node/
docker-compose down

docker container prune -f
docker image prune -f