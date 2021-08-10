#!/bin/bash

cd /home/ubuntu/storage-node/

GO_ENV=$(grep GO_ENV .env | cut -d '=' -f2)
echo "SIA_API_PASSWORD=$(aws ssm get-parameter --name /storage-node/$GO_ENV/SIA_API_PASSWORD --with-decryption --query Parameter.Value)" >> .env
echo "SIA_WALLET_PASSWORD=$(aws ssm get-parameter --name /storage-node/$GO_ENV/SIA_WALLET_PASSWORD --with-decryption --query Parameter.Value)" >> .env

docker-compose up -d

