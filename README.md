# Cachenator

[![Docker](https://github.com/adrianchifor/cachenator/workflows/Publish%20Docker/badge.svg)](https://github.com/adrianchifor/cachenator/actions?query=workflow%3A%22Publish+Docker%22) [![Go Report Card](https://goreportcard.com/badge/github.com/adrianchifor/cachenator)](https://goreportcard.com/report/github.com/adrianchifor/cachenator)

Distributed, sharded in-memory cache and proxy for S3.

Features:

- Horizontal scaling and clustering
- Read-through blob cache with TTL
- Batch parallel uploads
- Max memory limits with LRU evictions
- Fast cache keys invalidation
- Keys prefix pre-warming (soon)
- Batch parallel deletes (soon)
- Access multiple S3 endpoints (on-prem + AWS) (soon)

<img src="./docs/diagram.png">

---

## Run

```
$ docker run -it ghcr.io/adrianchifor/cachenator --help
Usage of /cachenator:
  -host string
    	Host/IP to identify self in peers list (default "localhost")
  -max-cache-size int
    	Max cache size in megabytes. If size goes above, oldest keys will be evicted (default 512)
  -max-multipart-memory int
    	Max memory in megabytes for /upload multipart form parsing (default 128)
  -peers string
    	Peers (default '', e.g. 'http://peer1:8080,http://peer2:8080')
  -port int
    	Server port (default 8080)
  -s3-download-concurrency int
    	Number of goroutines to spin up when downloading blob chunks from S3 (default 10)
  -s3-download-part-size int
    	Size in megabytes to request from S3 for each blob chunk (minimum 5) (default 5)
  -s3-endpoint string
    	Custom S3 endpoint URL (defaults to AWS)
  -s3-upload-concurrency int
    	Number of goroutines to spin up when uploading blob chunks to S3 (default 10)
  -s3-upload-part-size int
    	Buffer size in megabytes when uploading blob chunks to S3 (minimum 5) (default 5)
  -timeout int
    	Get blob timeout in milliseconds (default 5000)
  -ttl int
    	Blob time-to-live in cache in minutes (default 60)
  -verbose
    	Verbose logs
  -version
    	Version

$ docker run -d --name cache1 --network host -v $HOME/.aws/:/root/.aws:ro ghcr.io/adrianchifor/cachenator \
  --port 8080 \
  --peers http://localhost:8080,http://localhost:8081,http://localhost:8082

$ docker run -d --name cache2 --network host -v $HOME/.aws/:/root/.aws:ro ghcr.io/adrianchifor/cachenator \
  --port 8081 \
  --peers http://localhost:8080,http://localhost:8081,http://localhost:8082

$ docker run -d --name cache3 --network host -v $HOME/.aws/:/root/.aws:ro ghcr.io/adrianchifor/cachenator \
  --port 8082 \
  --peers http://localhost:8080,http://localhost:8081,http://localhost:8082
```

## Use

```bash
# Put files into bucket1:blob1 and bucket1:blob2
curl "http://localhost:8080/upload?bucket=bucket1" \
  -F "files=@/path/blob1" \
  -F "files=@/path/blob2"

# Put file into bucket1:folder/blob3
curl "http://localhost:8080/upload?bucket=bucket1&path=folder" \
  -F "files=@/path/blob3"

# First request takes longer as cache gets filled from S3
curl "http://localhost:8080/get?bucket=bucket1&key=blob1" > blob1

# 2nd+ requests served from memory
curl "http://localhost:8080/get?bucket=bucket1&key=blob1" > blob1

# Hitting any other node will get the hot blob from its owner and cache it as well before returning
curl "http://localhost:8081/get?bucket=bucket1&key=blob1" > blob1
curl "http://localhost:8082/get?bucket=bucket1&key=blob1" > blob1

# Remove blob from memory on all nodes
curl -X POST "http://127.0.0.1:8080/invalidate?bucket=bucket1&key=blob1"
```
