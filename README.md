# Falcon

[![Docker](https://github.com/adrianchifor/falcon/workflows/Publish%20Docker/badge.svg)](https://github.com/adrianchifor/falcon/actions?query=workflow%3A%22Publish+Docker%22) [![Go Report Card](https://goreportcard.com/badge/github.com/adrianchifor/falcon)](https://goreportcard.com/report/github.com/adrianchifor/falcon)

Distributed in-memory cache and proxy for S3.

Work in progress.

<img src="./docs/illustration.jpg" width="208" height="300">

---

## Build

```
make
```

## Run

```
./bin/falcon --help
Usage of ./bin/falcon:
  -bucket string
        S3 bucket name (required)
  -host string
        Host/IP to identify self in peers list (default "localhost")
  -k8s-discovery-id string
        Auto-discover peers on Kubernetes with label falcon-discovery-id=<k8s-discovery-id>
  -max-blob-size int
        Max blob size in megabytes (default 128)
  -peers string
        Peers (default '', e.g. 'http://peer1:8080,http://peer2:8080')
  -port int
        Server port (default 8080)
  -s3-endpoint string
        Custom S3 endpoint URL (defaults to AWS)
  -timeout int
        Get blob timeout in milliseconds (default 5000)
  -ttl int
        Blob time-to-live in cache in minutes (default 60)
  -verbose
        Verbose logs
  -version
        Version

./bin/falcon --bucket S3_BUCKET --peers http://localhost:8080,http://localhost:8081,http://localhost:8082 --port 8080
./bin/falcon --bucket S3_BUCKET --peers http://localhost:8080,http://localhost:8081,http://localhost:8082 --port 8081
./bin/falcon --bucket S3_BUCKET --peers http://localhost:8080,http://localhost:8081,http://localhost:8082 --port 8082
```

Multi-arch docker image is also available: `ghcr.io/adrianchifor/falcon:latest`

## Use

```bash
curl http://localhost:8080/upload \
  -F "files=@/path/blob1" \
  -F "files=@/path/blob2"

# First request takes longer as cache gets filled from S3
curl http://localhost:8080/get?key=blob1 > blob1

# 2nd+ requests served from memory
curl http://localhost:8080/get?key=blob1 > blob1

# Hitting any other node will get the hot blob from its owner and cache it as well before returning
curl http://localhost:8081/get?key=blob1 > blob1
curl http://localhost:8082/get?key=blob1 > blob1

# Remove blob from memory on all nodes
curl -X POST http://127.0.0.1:8080/invalidate?key=blob1
```
