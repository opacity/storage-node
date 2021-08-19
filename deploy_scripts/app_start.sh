#!/bin/bash

cd /home/ubuntu/storage-node/

GO_ENV=$(grep GO_ENV .env | cut -d '=' -f2)
echo "ADMIN_USER=$(aws ssm get-parameter --name /storage-node/$GO_ENV/ADMIN_USER --with-decryption --output text --query Parameter.Value)" >> .env
echo "ADMIN_PASSWORD=$(aws ssm get-parameter --name /storage-node/$GO_ENV/ADMIN_PASSWORD --with-decryption --output text --query Parameter.Value)" >> .env

docker-compose up -d
