# Falcon

[![Docker](https://github.com/adrianchifor/falcon/workflows/Publish%20Docker/badge.svg)](https://github.com/adrianchifor/falcon/actions?query=workflow%3A%22Publish+Docker%22) [![Go Report Card](https://goreportcard.com/badge/github.com/adrianchifor/falcon)](https://goreportcard.com/report/github.com/adrianchifor/falcon)

Distributed in-memory cache and proxy for S3.

Work in progress.

<img src="./docs/illustration.jpg" width="208" height="300">

---

## Run

```
$ docker run -it ghcr.io/adrianchifor/falcon --help
Usage of /falcon:
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

$ docker run -d --name falcon1 --network host -v $HOME/.aws/:/root/.aws:ro ghcr.io/adrianchifor/falcon \
  --port 8080 \
  --peers http://localhost:8080,http://localhost:8081,http://localhost:8082

$ docker run -d --name falcon2 --network host -v $HOME/.aws/:/root/.aws:ro ghcr.io/adrianchifor/falcon \
  --port 8081 \
  --peers http://localhost:8080,http://localhost:8081,http://localhost:8082

$ docker run -d --name falcon3 --network host -v $HOME/.aws/:/root/.aws:ro ghcr.io/adrianchifor/falcon \
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
