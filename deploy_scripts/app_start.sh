#!/bin/bash

cd /home/ubuntu/storage-node/

AWS_REGION=$(grep AWS_REGION .env | cut -d '=' -f2)

$(aws ecr get-login --region $AWS_REGION --no-include-email)

docker-compose up -d
