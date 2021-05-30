#!/bin/bash

cd /home/ubuntu/storage-node/
docker-compose up --build -d
docker image prune --all --force
