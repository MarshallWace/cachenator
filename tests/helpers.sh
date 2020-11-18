#!/usr/bin/env bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

CACHE="http://localhost:8080"
CACHE2="http://localhost:8081"
CACHE3="http://localhost:8082"

UPLOAD() { curl -s -o /dev/null -w '%{http_code}' "$@"; }
GET() { curl -s -o /tmp/cachenator_test -w '%{http_code}' "$1"; }
DELETE() { curl -X DELETE -s -o /dev/null -w '%{http_code}' "$1"; }
POST() { curl -X POST -s -o /dev/null -w '%{http_code}' "$1"; }

AWS() { aws --endpoint=http://localhost:4566 "$@"; }