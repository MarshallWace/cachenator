#!/usr/bin/env bats

load helpers.sh

# Get

@test "getting blobs from memory" {
  nodes=("$CACHE" "$CACHE2" "$CACHE3")
  for node in "${nodes[@]}"; do
    run GET "$node/get?bucket=$BUCKET&key=blob"
    [[ "$status" -eq 0 ]]
    [[ "$output" == "200" ]]
    [[ "$(SHA $DIR/blob)" == "$(SHA $TMP_BLOB)" ]]
  done

  run AWS_TRANSPARENT s3 cp s3://$BUCKET/blob_cached_transparent $TMP_BLOB
  [[ "$status" -eq 0 ]]
  [[ "$(SHA $DIR/blob)" == "$(SHA $TMP_BLOB)" ]]

  run AWS_TRANSPARENT s3api head-object s3://$BUCKET/blob_cached_transparent
  [[ "$status" -eq 0 ]]

  run GET "$CACHE/get?bucket=$BUCKET&key=cacheonwrite/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]
  [[ "$(SHA $DIR/blob)" == "$(SHA $TMP_BLOB)" ]]
}

@test "getting prewarmed blob from memory" {
  run GET "$CACHE/get?bucket=$BUCKET&key=somerandomblob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "404" ]]

  run GET "$CACHE/get?bucket=$BUCKET&key=folder/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]
  [[ "$(SHA $DIR/blob)" == "$(SHA $TMP_BLOB)" ]]
}

@test "checking if deleted blob was removed from memory" {
  run GET "$CACHE/get?bucket=$BUCKET&key=folder/subfolder/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "404" ]]

  run AWS_TRANSPARENT s3 cp s3://$BUCKET/blob_transparent $TMP_BLOB
  [[ "$status" -eq 1 ]]
}

# Upload

@test "trying to upload while S3 is down" {
  run POST "$CACHE/upload?bucket=$BUCKET" -F "files=@$DIR/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "500" ]]
}

# Invalidate

@test "invalidating blob from memory" {
  run POST "$CACHE/invalidate?bucket=$BUCKET"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "400" ]]

  run POST "$CACHE/invalidate"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "400" ]]

  run POST "$CACHE/invalidate?bucket=$BUCKET&key=blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run GET "$CACHE/get?bucket=$BUCKET&key=blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "404" ]]
}

# Metrics

@test "checking if cluster is exposing prometheus metrics" {
  nodes=("$CACHE_METRICS" "$CACHE2_METRICS" "$CACHE3_METRICS")
  for node in "${nodes[@]}"; do
    run GET "$node/metrics"
    [[ "$status" -eq 0 ]]
    [[ "$output" == "200" ]]
    [[ "$(head -n1 $TMP_BLOB)" == "# HELP cachenator_cache_bytes Current (main/hot) cache bytes" ]]
  done
}