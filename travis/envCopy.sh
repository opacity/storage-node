#!/bin/bash
set -ev
sudo sed -i -e 's/AWS_REGION=TRAVIS/AWS_REGION='${AWS_REGION}'/' .env
