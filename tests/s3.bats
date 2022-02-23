#!/usr/bin/env bats

load helpers.sh

# Upload

@test "uploading blob to test bucket" {
  run POST "$CACHE/upload?bucket=$BUCKET"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "400" ]]

  run POST "$CACHE/upload"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "400" ]]

  run POST "$CACHE/upload?bucket=$BUCKET" -F "files=@$DIR/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run AWS s3 ls s3://$BUCKET/blob
  [[ "$status" -eq 0 ]]

  run AWS_TRANSPARENT s3 cp $DIR/blob s3://$BUCKET/blob_transparent
  [[ "$status" -eq 0 ]]

  # We upload a second blob to check later if its cached when S3 is down
  run AWS_TRANSPARENT s3 cp $DIR/blob s3://$BUCKET/blob_cached_transparent
  [[ "$status" -eq 0 ]]

  run AWS_TRANSPARENT s3 ls s3://$BUCKET/blob_transparent
  [[ "$status" -eq 0 ]]

  run AWS s3 ls s3://$BUCKET/blob_transparent
  [[ "$status" -eq 0 ]]
}

@test "uploading blob to test bucket with paths" {
  run POST "$CACHE/upload?bucket=$BUCKET&path=folder" -F "files=@$DIR/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run AWS s3 ls s3://$BUCKET/folder/blob
  [[ "$status" -eq 0 ]]

  run POST "$CACHE/upload?bucket=$BUCKET&path=folder/subfolder" -F "files=@$DIR/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run AWS s3 ls s3://$BUCKET/folder/subfolder/blob
  [[ "$status" -eq 0 ]]

  run POST "$CACHE2/upload?bucket=$BUCKET&path=cacheonwrite" -F "files=@$DIR/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run AWS s3 ls s3://$BUCKET/cacheonwrite/blob
  [[ "$status" -eq 0 ]]
}

# Get

@test "getting blob from test bucket" {
  run GET "$CACHE/get?bucket=$BUCKET"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "400" ]]

  run GET "$CACHE/get"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "400" ]]

  run GET "$CACHE/get?bucket=$BUCKET&key=somerandomblob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "404" ]]

  run GET "$CACHE/get?bucket=$BUCKET&key=blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]
  [[ "$(SHA $DIR/blob)" == "$(SHA $TMP_BLOB)" ]]

  run GET "$CACHE/get?bucket=$BUCKET&key=folder/subfolder/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]
  [[ "$(SHA $DIR/blob)" == "$(SHA $TMP_BLOB)" ]]

  run AWS_TRANSPARENT s3 cp s3://$BUCKET/blob_cached_transparent $TMP_BLOB
  [[ "$status" -eq 0 ]]
  [[ "$(SHA $DIR/blob)" == "$(SHA $TMP_BLOB)" ]]

  run AWS_TRANSPARENT s3api head-object --bucket=$BUCKET --key=blob_cached_transparent
  [[ "$status" -eq 0 ]]
}

# List

@test "listing keys from test bucket" {
  run GET "$CACHE/list"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "400" ]]

  run GET "$CACHE/list?bucket=$BUCKET&prefix=somerandompath"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "404" ]]

  run GET "$CACHE/list?bucket=$BUCKET"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]
  [[ "$(cat $TMP_BLOB | jq '.keys | length')" == "6" ]]

  run GET "$CACHE/list?bucket=$BUCKET&prefix=folder"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]
  [[ "$(cat $TMP_BLOB | jq '.keys | length')" == "2" ]]

  run AWS_TRANSPARENT s3 ls s3://$BUCKET/
  [[ "$status" -eq 0 ]]
}

# Delete

@test "deleting blob from test bucket" {
  run DELETE "$CACHE/delete?bucket=$BUCKET"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "400" ]]

  run DELETE "$CACHE/delete"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "400" ]]

  run DELETE "$CACHE/delete?bucket=$BUCKET&key=folder/subfolder/blob"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run AWS s3 ls s3://$BUCKET/folder/subfolder/blob
  [[ "$status" -eq 1 ]]

  run AWS_TRANSPARENT s3 rm s3://$BUCKET/blob_transparent
  [[ "$status" -eq 0 ]]

  run AWS s3 ls s3://$BUCKET/blob_transparent
  [[ "$status" -eq 1 ]]
}

@test "deleting blob prefix from test bucket" {
  run DELETE "$CACHE/delete?bucket=$BUCKET&prefix=folder/subfolder"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]

  run AWS s3 ls s3://$BUCKET/folder/subfolder
  [[ "$status" -eq 1 ]]
}

# Prewarm

@test "prewarming blob prefix from test bucket" {
  run POST "$CACHE/prewarm?bucket=$BUCKET"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "400" ]]

  run POST "$CACHE/prewarm"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "400" ]]

  run POST "$CACHE/prewarm?bucket=$BUCKET&prefix=somerandompath"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "404" ]]

  run POST "$CACHE/prewarm?bucket=$BUCKET&prefix=folder"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "200" ]]
}