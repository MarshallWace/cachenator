#!/usr/bin/env bash

# make test

set -u
set -eE

# Get directory of script no matter where it's called from
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

source $DIR/helpers.sh
trap cleanup ERR

echo -e "\nRunning AWS S3 localstack"
docker run -d --name localstack-s3 -e SERVICES=s3 -p 4566:4566 localstack/localstack:0.12.2
echo -e "Waiting 30s for AWS S3 localstack to be ready ..."
sleep 30

echo -e "Creating test S3 bucket"
aws --endpoint=$AWS_ENDPOINT s3api create-bucket --bucket $BUCKET

echo -e "\nRunning cachenator cluster"
run_cachenator

echo -e "\nRunning S3 tests"
bats $DIR/s3.bats

echo -e "Stopping AWS S3 localstack"
docker rm -f localstack-s3 >/dev/null 2>&1

echo -e "\nRunning memory tests"
bats $DIR/memory.bats

cleanup
