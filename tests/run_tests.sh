#!/usr/bin/env bash

# make test

set -u
set -eE

[[ -z $(which bats) ]] && echo "bats not found, install: https://github.com/bats-core/bats-core#installation" && exit 1
[[ -z $(which docker) ]] && echo "docker not found, install: https://docs.docker.com/get-docker/" && exit 1
[[ -z $(which curl) ]] && echo "curl not found, install: https://curl.se/download.html" && exit 1
[[ -z $(which aws) ]] && echo "aws not found, install: pip3 install --user awscli" && exit 1

function cleanup() {
  echo "Cleaning up cachenator processes"
  for process in $(pgrep cachenator); do
    kill "$process"
  done

  echo "Cleaning up AWS S3 localstack"
  docker rm -f localstack-s3 >/dev/null 2>&1 || true

  echo "Cleaning up /tmp"
  rm -f /tmp/cachenator_test

  echo "Done"
}

trap cleanup ERR

# Get directory of script no matter where it's called from
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

echo -e "\nRunning AWS S3 localstack"
docker run -d --name localstack-s3 -e SERVICES=s3 -p 4566:4566 localstack/localstack:0.12.2
echo -e "Waiting 30s for AWS S3 localstack to be ready ..."
sleep 30

echo -e "Creating test S3 bucket"
aws --endpoint=http://localhost:4566 s3api create-bucket --bucket test

echo -e "\nRunning cachenator cluster"
$DIR/../bin/cachenator -port 8080 -peers localhost:8080,localhost:8081,localhost:8082 \
  -s3-endpoint http://localhost:4566 -s3-force-path-style >/dev/null 2>&1 &
$DIR/../bin/cachenator -port 8081 -peers localhost:8080,localhost:8081,localhost:8082 \
  -s3-endpoint http://localhost:4566 -s3-force-path-style >/dev/null 2>&1 &
$DIR/../bin/cachenator -port 8082 -peers localhost:8080,localhost:8081,localhost:8082 \
  -s3-endpoint http://localhost:4566 -s3-force-path-style >/dev/null 2>&1 &

echo -e "\nRunning S3 tests"
bats $DIR/s3.bats

echo -e "Stopping AWS S3 localstack"
docker rm -f localstack-s3 >/dev/null 2>&1

echo -e "\nRunning memory tests"
bats $DIR/memory.bats

cleanup


