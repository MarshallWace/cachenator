# falcon
Distributed in-memory cache and proxy for S3

Work in progress

## Build

```
make
```

## Run

```
./bin/falcon --bucket S3_BUCKET --peers http://localhost:8080,http://localhost:8081 --port 8080
./bin/falcon --bucket S3_BUCKET --peers http://localhost:8080,http://localhost:8081 --port 8081
```

## Use

```bash
curl -X POST http://127.0.0.1:8080/upload \
  -F "files=@/path/blob1" \
  -F "files=@/path/blob2" \
  -H "Content-Type: multipart/form-data"

# First request takes longer as cache gets filled from S3
curl http://127.0.0.1:8080/get?key=blob1 > blob1

# 2nd+ request super fast (99th percentile ~1ms)
curl http://127.0.0.1:8080/get?key=blob1 > blob1

# Cache is sharded so hitting 2nd node will just pull the blob from the 1st node
curl http://127.0.0.1:8081/get?key=blob1 > blob1
```