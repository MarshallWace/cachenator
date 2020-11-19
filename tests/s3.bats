#!/usr/bin/env bats

load helpers.sh

# Upload

@test "uploading blob to test bucket" {
  run UPLOAD "$CACHE/upload?bucket=$BUCKET" -F "files=@$DIR/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run AWS s3 ls s3://$BUCKET/blob
  [[ "$status" -eq 0 ]]
}

@test "uploading blob to test bucket with paths" {
  run UPLOAD "$CACHE/upload?bucket=$BUCKET&path=folder" -F "files=@$DIR/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run AWS s3 ls s3://$BUCKET/folder/blob
  [[ "$status" -eq 0 ]]

  run UPLOAD "$CACHE/upload?bucket=$BUCKET&path=folder/subfolder" -F "files=@$DIR/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run AWS s3 ls s3://$BUCKET/folder/subfolder/blob
  [[ "$status" -eq 0 ]]
}

# Delete

@test "deleting blob from test bucket" {
  run DELETE "$CACHE/delete?bucket=$BUCKET&key=folder/subfolder/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run AWS s3 ls s3://$BUCKET/folder/subfolder/blob
  [ "$status" -eq 1 ]
}

@test "deleting blob prefix from test bucket" {
  run DELETE "$CACHE/delete?bucket=$BUCKET&prefix=folder/subfolder"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run AWS s3 ls s3://$BUCKET//folder/subfolder
  [ "$status" -eq 1 ]
}

# Get

@test "getting blob from test bucket" {
  run GET "$CACHE/get?bucket=$BUCKET&key=blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]
  [[ "$(SHA $DIR/blob)" == "$(SHA $TMP_BLOB)" ]]
}

# Prewarm

@test "prewarming blob prefix from test bucket" {
  run POST "$CACHE/prewarm?bucket=$BUCKET&prefix=folder"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]
}