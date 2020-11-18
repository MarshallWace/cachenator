#!/usr/bin/env bats

load helpers.sh

# Upload

@test "uploading blob to test bucket" {
  run UPLOAD "$CACHE/upload?bucket=test" -F "files=@$DIR/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run AWS s3 ls s3://test/blob
  [[ "$status" -eq 0 ]]
}

@test "uploading blob to test bucket with paths" {
  run UPLOAD "$CACHE/upload?bucket=test&path=folder" -F "files=@$DIR/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run AWS s3 ls s3://test/folder/blob
  [[ "$status" -eq 0 ]]

  run UPLOAD "$CACHE/upload?bucket=test&path=folder/subfolder" -F "files=@$DIR/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run AWS s3 ls s3://test/folder/subfolder/blob
  [[ "$status" -eq 0 ]]
}

# Delete

@test "deleting blob from test bucket" {
  run DELETE "$CACHE/delete?bucket=test&key=folder/subfolder/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run AWS s3 ls s3://test/folder/subfolder/blob
  [ "$status" -eq 1 ]
}

@test "deleting blob prefix from test bucket" {
  run DELETE "$CACHE/delete?bucket=test&prefix=folder/subfolder"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run AWS s3 ls s3://test//folder/subfolder
  [ "$status" -eq 1 ]
}

# Get

@test "getting blob from test bucket" {
  run GET "$CACHE/get?bucket=test&key=blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  local_blob_hash=$(sha256sum $DIR/blob | awk '{print $1}')
  pulled_blob_hash=$(sha256sum /tmp/cachenator_test | awk '{print $1}')
  [[ "$local_blob_hash" == "$pulled_blob_hash" ]]
}

# Prewarm

@test "prewarming blob prefix from test bucket" {
  run POST "$CACHE/prewarm?bucket=test&prefix=folder"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]
}