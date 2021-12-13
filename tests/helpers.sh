#!/usr/bin/env bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

BUCKET="test"
TMP_BLOB="/tmp/cachenator_test"

CACHE="http://localhost:8080"
CACHE2="http://localhost:8081"
CACHE3="http://localhost:8082"
CACHE_METRICS="http://localhost:9095"
CACHE2_METRICS="http://localhost:9096"
CACHE3_METRICS="http://localhost:9097"
AWS_ENDPOINT="http://localhost:4566"

POST() { curl -X POST -s -o /dev/null -w '%{http_code}' "$@"; }
GET() { curl -s -o $TMP_BLOB -w '%{http_code}' "$1"; }
DELETE() { curl -X DELETE -s -o /dev/null -w '%{http_code}' "$1"; }
AWS() { aws --endpoint=$AWS_ENDPOINT "$@"; }
AWS_TRANSPARENT() { aws --endpoint=$CACHE "$@"; }

SHA() { echo $(sha256sum "$1" | awk '{print $1}'); }

try_command() {
  command -v "$1" >/dev/null 2>&1
}

try_command bats || {
  echo "bats not found, install: https://github.com/bats-core/bats-core#installation"
  exit 1
}
try_command docker || {
  try_command podman || {
    echo "docker not found, install: https://docs.docker.com/get-docker/"
    exit 1
  }
  docker() { podman "$@"; }
}
try_command curl || {
  echo "curl not found, install: https://curl.se/download.html"
  exit 1
}
try_command jq || {
  echo "jq not found, install: https://stedolan.github.io/jq/download/"
  exit 1
}
try_command aws || {
  echo "aws not found, install: pip3 install --user awscli"
  exit 1
}

run_cachenator() {
  export AWS_REGION="eu-west-2"
  $DIR/../bin/cachenator -port 8080 -metrics-port 9095 -peers $CACHE,$CACHE2,$CACHE3 \
    -s3-endpoint $AWS_ENDPOINT -s3-force-path-style -s3-transparent-api >/dev/null 2>&1 &
  $DIR/../bin/cachenator -port 8081 -metrics-port 9096 -peers $CACHE,$CACHE2,$CACHE3 \
    -s3-endpoint $AWS_ENDPOINT -s3-force-path-style -cache-on-write >/dev/null 2>&1 &
  $DIR/../bin/cachenator -port 8082 -metrics-port 9097 -peers $CACHE,$CACHE2,$CACHE3 \
    -s3-endpoint $AWS_ENDPOINT -s3-force-path-style >/dev/null 2>&1 &
}

run_cachenator_jwt() {
  export AWS_REGION="eu-west-2"
  $DIR/../bin/cachenator -port 8080 -jwt-rsa-publickey-path $DIR/pubkey.crt \
    -s3-endpoint $AWS_ENDPOINT -s3-force-path-style >/dev/null 2>&1 &
}

cleanup() {
  echo "Cleaning up cachenator processes"
  pgrep cachenator | xargs kill

  echo "Cleaning up AWS S3 localstack"
  docker rm -f localstack-s3 >/dev/null 2>&1 || true

  echo "Cleaning up /tmp"
  rm -f $TMP_BLOB

  echo "Done"
}