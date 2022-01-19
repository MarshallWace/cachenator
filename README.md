# Cachenator

[![Docker](https://github.com/MarshallWace/cachenator/workflows/Publish%20Docker/badge.svg)](https://github.com/MarshallWace/cachenator/actions?query=workflow%3A%22Publish+Docker%22) [![Go Report Card](https://goreportcard.com/badge/github.com/MarshallWace/cachenator)](https://goreportcard.com/report/github.com/MarshallWace/cachenator)

Distributed, sharded in-memory cache and proxy for S3.

Features:

- Horizontal scaling and clustering
- Read-through blob cache with TTL
- Transparent S3 usage (awscli or SDKs)
- Batch parallel uploads and deletes
- Max memory limits with LRU evictions
- Fast cache keys invalidation
- Async cache pre-warming (with keys prefix)
- Cache on write
- Prometheus metrics
- Access multiple S3 endpoints (on-prem + AWS) (soon)

<img src="./docs/diagram.png">

---

## Run

```
$ docker run -it ghcr.io/marshallwace/cachenator --help
Usage of /cachenator:
  -cache-on-write
    	Enable automatic caching on uploads (default false)
  -disable-http-metrics
    	Disable HTTP metrics (req/s, latency) when expecting high path cardinality (default false)
  -host string
    	Host/IP to identify self in peers list (default "localhost")
  -log-level string
    	Logging level (info, debug, error, warn) (default "info")
  -max-cache-size int
    	Max cache size in megabytes. If size goes above, oldest keys will be evicted (default 512)
  -max-multipart-memory int
    	Max memory in megabytes for /upload multipart form parsing (default 128)
  -metrics-port int
    	Prometheus metrics port (default 9095)
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
  -s3-force-path-style
    	Force S3 path bucket addressing (endpoint/bucket/key vs. bucket.endpoint/key) (default false)
  -s3-transparent-api
    	Enable transparent S3 API for usage from awscli or SDKs (default false)
  -s3-upload-concurrency int
    	Number of goroutines to spin up when uploading blob chunks to S3 (default 10)
  -s3-upload-part-size int
    	Buffer size in megabytes when uploading blob chunks to S3 (minimum 5) (default 5)
  -timeout int
    	Get blob timeout in milliseconds (default 5000)
  -ttl int
    	Blob time-to-live in cache in minutes (0 to never expire) (default 60)
  -version
    	Version

$ docker run -d --name cache1 --network host -v $HOME/.aws/:/root/.aws:ro ghcr.io/marshallwace/cachenator \
  --port 8080 --metrics-port 9095 \
  --peers http://localhost:8080,http://localhost:8081,http://localhost:8082

$ docker run -d --name cache2 --network host -v $HOME/.aws/:/root/.aws:ro ghcr.io/marshallwace/cachenator \
  --port 8081 --metrics-port 9096 \
  --peers http://localhost:8080,http://localhost:8081,http://localhost:8082

$ docker run -d --name cache3 --network host -v $HOME/.aws/:/root/.aws:ro ghcr.io/marshallwace/cachenator \
  --port 8082 --metrics-port 9097 \
  --peers http://localhost:8080,http://localhost:8081,http://localhost:8082
```

## Use

### REST API

```bash
##########
# Upload #
##########

curl "http://localhost:8080/upload?bucket=bucket1" \
  -F "files=@blob1"

curl "http://localhost:8080/upload?bucket=bucket1&path=folder" \
  -F "files=@blob2" \
  -F "files=@blob3" \
  -F "files=@blob4"

#######
# Get #
#######

# First request fills cache from S3
curl "http://localhost:8080/get?bucket=bucket1&key=blob1" > blob1

# 2nd+ requests served from memory
curl "http://localhost:8080/get?bucket=bucket1&key=blob1" > blob1

# Hitting other nodes will get the blob from the shard owner and cache it as well before returning
curl "http://localhost:8081/get?bucket=bucket1&key=blob1" > blob1
curl "http://localhost:8082/get?bucket=bucket1&key=blob1" > blob1

########
# List #
########

curl "http://localhost:8080/list?bucket=bucket1&prefix=folder" | jq '.keys'

############
# Pre-warm #
############

# Pre-pull in the background and cache keys 'folder/[blob2/blob3/blob4]'
curl -XPOST "http://localhost:8080/prewarm?bucket=bucket1&prefix=folder/blob"

# Served straight from memory
curl "http://localhost:8080/get?bucket=bucket1&key=folder/blob2" > blob2

##############
# Invalidate #
##############

# Remove blob1 from memory on all nodes
curl -XPOST "http://localhost:8080/invalidate?bucket=bucket1&key=blob1"

##########
# Delete #
##########

# Delete only blob1 from S3
curl -XDELETE "http://localhost:8080/delete?bucket=bucket1&key=blob1"

# Delete keys 'folder/[blob2/blob3/blob4]' from S3
curl -XDELETE "http://localhost:8080/delete?bucket=bucket1&prefix=folder/blob"

###########
# Metrics #
###########

curl "http://localhost:9095/metrics"
```

### Transparent S3 usage (awscli or SDKs)

```bash
docker run -d --name transparent_cache --network host -v $HOME/.aws/:/root/.aws:ro \
  ghcr.io/marshallwace/cachenator --port 8083 -s3-transparent-api

aws --endpoint=http://localhost:8083 s3 cp blob1 s3://bucket1/blob1
upload: blob1 to s3://bucket1/blob1

aws --endpoint=http://localhost:8083 s3 ls s3://bucket1
2021-10-15 20:45:13     333516 blob1

aws --endpoint=http://localhost:8083 s3 cp s3://bucket1/blob1 /tmp/blob.png
download: s3://bucket1/blob1 to /tmp/blob.png

aws --endpoint=http://localhost:8083 s3 rm s3://bucket1/blob1
delete: s3://bucket1/blob1

aws --endpoint=http://localhost:8083 s3 ls s3://bucket1
# Empty
```

### JWT auth

This feature will enable authentication on all endpoints (except /healthz) and is helpful for clients that require temporary access to S3 or can't get dedicated S3 creds. This is also helpful for simulating the [AWS signed URLs](https://docs.aws.amazon.com/AmazonS3/latest/userguide/ShareObjectPreSignedURL.html) functionality for custom S3 providers like [Pure Flashblade](https://www.purestorage.com/uk/products/file-and-object/flashblade.html).

An example use case looks like:
- client requires read access to an S3 blob
- client authenticates with an oauth2/kerberos/custom auth provider
- auth provider issues a temporary RS256 JWT token with a payload like:
  ```
  {
    "exp": <unix timestamp now+5min>,
    "iss": "<auth provider>", # optional
    "aud": "cachenator,       # optional
    "action": "READ",
    "bucket": "mybucket",     # required, or set to "" to allow all
    "prefix": "myobject",     # required, or set to "" to allow all
  }
  ```
- client passes JWT token to cachenator endpoint in the Authorization header
- cachenator validates JWT token, action, issuer, audience, bucket and prefix and responds with blob

#### JWT usage

To enable JWT auth on all endpoints, pass in the `-jwt-rsa-publickey-path` flag. The JWT token issuer will need to have the equivalent RSA private key to sign the tokens, cachenator just needs the public key to validate the signature.

```bash
docker run -d --network host -v $HOME/.aws/:/root/.aws:ro -v $(pwd):/certs \
  ghcr.io/marshallwace/cachenator -jwt-rsa-publickey-path /certs/publickey.crt

curl "http://localhost:8080/get?bucket=test&key=blob" \
  -H "Authorization: Bearer <JWT token>" > blob
```

To also validate standard claims like issuer and audience:

```bash
docker run -d --network host -v $HOME/.aws/:/root/.aws:ro -v $(pwd):/certs \
  ghcr.io/marshallwace/cachenator -jwt-rsa-publickey-path /certs/publickey.crt \
  -jwt-issuer <auth provider> -jwt-audience cachenator
```
