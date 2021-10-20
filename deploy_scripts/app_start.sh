#!/bin/bash

start=$(date +%s)

cd /home/ubuntu/storage-node/

AWS_REGION=$(grep AWS_REGION .env | cut -d '=' -f2)
GO_ENV=$(grep GO_ENV .env | cut -d '=' -f2)
VERSION=$(grep VERSION .env | cut -d '=' -f2)

echo "ADMIN_USER=$(aws ssm get-parameter --name /storage-node/$GO_ENV/ADMIN_USER --with-decryption --output text --query Parameter.Value)" >> .env
echo "ADMIN_PASSWORD=$(aws ssm get-parameter --name /storage-node/$GO_ENV/ADMIN_PASSWORD --with-decryption --output text --query Parameter.Value)" >> .env

$(aws ecr get-login --region $AWS_REGION --no-include-email)

docker-compose up -d

now=$(date +%s)
sentry-cli releases deploys "$VERSION" new -e $GO_ENV -t $((now-start)) -u $(aws ssm get-parameter --name /storage-node/$GO_ENV/FRONTEND_URL --with-decryption --output text --query Parameter.Value)
sentry-cli releases finalize "$VERSION"
