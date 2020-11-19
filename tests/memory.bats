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
}

@test "getting prewarmed blob from memory" {
  run GET "$CACHE/get?bucket=$BUCKET&key=folder/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]
  [[ "$(SHA $DIR/blob)" == "$(SHA $TMP_BLOB)" ]]
}

# Invalidate

@test "invalidating blob from memory" {
  run POST "$CACHE/invalidate?bucket=$BUCKET&key=blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run GET "$CACHE/get?bucket=$BUCKET&key=blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "404" ]]
}