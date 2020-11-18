#!/usr/bin/env bats

load helpers.sh

# Get

@test "getting blobs from memory" {
  nodes=("$CACHE" "$CACHE2" "$CACHE3")
  for node in "${nodes[@]}"; do
    run GET "$node/get?bucket=test&key=blob"
    [[ "$status" -eq 0 ]]
    [[ "$output" == "200" ]]

    local_blob_hash=$(sha256sum $DIR/blob | awk '{print $1}')
    pulled_blob_hash=$(sha256sum /tmp/cachenator_test | awk '{print $1}')
    [[ "$local_blob_hash" == "$pulled_blob_hash" ]]
  done
}

@test "getting prewarmed blob from memory" {
  run GET "$CACHE/get?bucket=test&key=folder/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  local_blob_hash=$(sha256sum $DIR/blob | awk '{print $1}')
  pulled_blob_hash=$(sha256sum /tmp/cachenator_test | awk '{print $1}')
  [[ "$local_blob_hash" == "$pulled_blob_hash" ]]
}

# Invalidate

@test "invalidating blob from memory" {
  run POST "$CACHE/invalidate?bucket=test&key=blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run GET "$CACHE/get?bucket=test&key=blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "404" ]]
}