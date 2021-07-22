// Copyright 2021 Adrian Chifor, Marshall Wace
// SPDX-FileCopyrightText: 2021 Marshall Wace <opensource@mwam.com>
// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/adrianchifor/go-parallel"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gin-gonic/gin"
	"github.com/mailgun/groupcache/v2"
	log "github.com/sirupsen/logrus"
)

var (
	s3Endpoint          string
	s3ForcePathStyle    bool
	s3Client            s3iface.S3API
	uploadPartSize      int64
	uploadConcurrency   int
	s3Uploader          s3manager.Uploader
	downloadPartSize    int64
	downloadConcurrency int
	s3Downloader        s3manager.Downloader
)

func initS3() {
	s3Session, err := session.NewSession(&aws.Config{
		Endpoint:         aws.String(s3Endpoint),
		S3ForcePathStyle: aws.Bool(s3ForcePathStyle),
	})
	if err != nil {
		log.Fatalf("Failed to initialize S3 session: %v", err)
	}
	s3Client = s3iface.S3API(s3.New(s3Session))

	s3Uploader = *s3manager.NewUploader(s3Session, func(u *s3manager.Uploader) {
		u.PartSize = uploadPartSize * 1024 * 1024
		u.Concurrency = uploadConcurrency
	})
	s3Downloader = *s3manager.NewDownloader(s3Session, func(d *s3manager.Downloader) {
		d.PartSize = downloadPartSize * 1024 * 1024
		d.Concurrency = downloadConcurrency
		d.BufferProvider = s3manager.NewPooledBufferedWriterReadFromProvider(5 * 1024 * 1024)
	})
}

func restS3Upload(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		log.Errorf("Failed to parse multipart form: %v", err)
		c.JSON(400, gin.H{"error": "Expecting a multipart form"})
		return
	}
	if _, found := form.File["files"]; !found {
		c.JSON(400, gin.H{"error": "'files' not found in multipart form"})
		return
	}
	bucket := strings.TrimSpace(c.Query("bucket"))
	if bucket == "" {
		c.JSON(400, gin.H{"error": "'bucket' not found in querystring parameters"})
		return
	}
	path := strings.TrimSpace(c.Query("path"))
	// Add trailing / if doesn't exist and path is set
	if path != "" && !strings.HasSuffix(path, "/") {
		path = fmt.Sprintf("%s/", path)
	}

	files := form.File["files"]

	uploadPool := parallel.SmallJobPool()
	defer uploadPool.Close()

	uploadsFailed := []string{}
	uploadsFailedMutex := &sync.Mutex{}

	for _, file := range files {
		file := file

		uploadPool.AddJob(func() {
			key := file.Filename
			fullKey := fmt.Sprintf("%s%s", path, key)
			log.Debugf("Uploading '%s#%s' to S3", bucket, fullKey)

			body, err := file.Open()
			if err != nil {
				log.Errorf("Failed to read '%s' file when trying to upload to S3: %v", key, err)
				uploadsFailedMutex.Lock()
				defer uploadsFailedMutex.Unlock()
				uploadsFailed = append(uploadsFailed, fullKey)
				return
			}
			defer body.Close()

			_, err = s3Uploader.Upload(&s3manager.UploadInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(fullKey),
				Body:   body,
			})
			if err != nil {
				log.Errorf("Failed to upload '%s' to S3 bucket '%s': %v", fullKey, bucket, err)
				uploadsFailedMutex.Lock()
				defer uploadsFailedMutex.Unlock()
				uploadsFailed = append(uploadsFailed, fullKey)
				return
			}
			log.Debugf("Upload to S3 done for '%s#%s'", bucket, fullKey)

			// Invalidate uploaded blob if in-memory
			go cacheInvalidate(bucket, fullKey)
		})
	}

	err = uploadPool.Wait()
	if err != nil {
		log.Error(err)
		c.JSON(500, gin.H{
			"error": "Internal error, check server logs",
		})
		return
	}

	if len(uploadsFailed) > 0 {
		c.JSON(500, gin.H{
			"error":         "Failed to upload some blobs",
			"uploadsFailed": uploadsFailed,
		})
		return
	}

	c.JSON(200, gin.H{
		"message": fmt.Sprintf("Uploaded %d object(s) to S3 bucket '%s'", len(files), bucket),
		"error":   "",
	})
}

func transparentS3Put(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	_, err := s3Uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   c.Request.Body,
	})
	if err != nil {
		log.Errorf("Failed to upload '%s' to S3 bucket '%s': %v", key, bucket, err)
		c.String(500, "")
		return
	}

	// Invalidate uploaded blob if in-memory
	go cacheInvalidate(bucket, key)

	c.String(200, "")
}

func transparentS3Get(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	cacheKey := constructCacheKey(bucket, key)
	log.Debugf("Checking cache for '%s'", cacheKey)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(timeout))
	defer cancel()

	var cacheView groupcache.ByteView
	if err := cacheGroup.Get(ctx, cacheKey, groupcache.ByteViewSink(&cacheView)); err != nil {
		c.String(404, "")
		return
	}

	c.DataFromReader(200, int64(cacheView.Len()), "application/octet-stream", cacheView.Reader(), nil)
}

func restS3Delete(c *gin.Context) {
	bucket := strings.TrimSpace(c.Query("bucket"))
	if bucket == "" {
		c.JSON(400, gin.H{"error": "'bucket' not found in querystring parameters"})
		return
	}
	key := strings.TrimSpace(c.Query("key"))
	prefix := strings.TrimSpace(c.Query("prefix"))
	if key == "" && prefix == "" {
		c.JSON(400, gin.H{"error": "'key' or 'prefix' not found in querystring parameters"})
		return
	}
	if key != "" && prefix != "" {
		c.JSON(400, gin.H{"error": "Only provide one of 'key' or 'prefix' in querystring parameters"})
		return
	}

	if key != "" {
		err := s3Delete(bucket, key)
		if err != nil {
			msg := fmt.Sprintf("Failed to delete '%s#%s' from S3: %v", bucket, key, err)
			log.Errorf(msg)
			c.JSON(500, gin.H{"error": msg})
			return
		}

		c.JSON(200, gin.H{
			"message": fmt.Sprintf("Deleted '%s#%s' from S3", bucket, key),
			"error":   "",
		})
	} else {
		log.Debugf("Deleting prefix '%s#%s' from S3", bucket, prefix)
		keysToDelete, _ := s3ListKeys(bucket, prefix, "")
		iter := s3manager.NewDeleteListIterator(s3Client, &s3.ListObjectsInput{
			Bucket: aws.String(bucket),
			Prefix: aws.String(prefix),
		})
		if err := s3manager.NewBatchDeleteWithClient(s3Client).Delete(aws.BackgroundContext(), iter); err != nil {
			msg := fmt.Sprintf("Failed to batch delete '%s#%s' from S3: %v", bucket, prefix, err)
			log.Errorf(msg)
			c.JSON(500, gin.H{"error": msg})
			return
		}
		msg := fmt.Sprintf("Deleted object(s) with prefix '%s' from S3 bucket '%s'", prefix, bucket)
		log.Debugf(msg)

		if keysToDelete != nil {
			// Invalidate deleted blobs if in-memory
			log.Debugf("Invalidating keys from cache with prefix '%s' for bucket '%s'", prefix, bucket)
			for _, key := range keysToDelete {
				key := key
				go cacheInvalidate(bucket, key)
			}
		}

		c.JSON(200, gin.H{
			"message": msg,
			"error":   "",
		})
	}
}

func transparentS3Delete(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	err := s3Delete(bucket, key)
	if err != nil {
		log.Errorf("Failed to delete '%s' from S3 bucket '%s': %v", key, bucket, err)
		c.String(500, "")
		return
	}
	c.String(204, "")
}

func s3Delete(bucket string, key string) error {
	_, err := s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}

	s3Client.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	log.Debugf(fmt.Sprintf("Deleted '%s#%s' from S3", bucket, key))

	// Invalidate deleted blob if in-memory
	go cacheInvalidate(bucket, key)

	return nil
}

func restS3List(c *gin.Context) {
	bucket := strings.TrimSpace(c.Query("bucket"))
	if bucket == "" {
		c.JSON(400, gin.H{"error": "'bucket' not found in querystring parameters"})
		return
	}
	prefix := strings.TrimSpace(c.Query("prefix"))
	delimiter := strings.TrimSpace(c.Query("delimiter"))

	keys, err := s3ListKeys(bucket, prefix, delimiter)
	if err != nil {
		msg := fmt.Sprintf("Failed to list keys in S3 bucket '%s': %v", bucket, err)
		log.Errorf(msg)
		c.JSON(500, gin.H{"error": msg})
		return
	}

	status := 200
	if len(keys) == 0 {
		status = 404
	}
	c.JSON(status, gin.H{"keys": keys})
}

func transparentS3ListBuckets(c *gin.Context) {
	s3buckets, err := s3Client.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		c.XML(500, Error{"InternalError", fmt.Sprintf("Failed to list buckets from S3: %v", err)})
		return
	}

	buckets := []Bucket{}
	for _, bucket := range s3buckets.Buckets {
		buckets = append(buckets, Bucket{*bucket.Name, *bucket.CreationDate})
	}
	c.XML(200, ListAllMyBucketsResult{buckets, Owner{
		*s3buckets.Owner.DisplayName,
		*s3buckets.Owner.ID,
	}})
}

func transparentS3ListObjects(c *gin.Context) {
	bucket := c.Param("bucket")
	prefix := strings.TrimSpace(c.Query("prefix"))
	delimiter := strings.TrimSpace(c.Query("delimiter"))

	s3objects, s3commonPrefixes, err := s3ListObjects(bucket, prefix, delimiter)
	if err != nil {
		c.XML(500, Error{"InternalError", fmt.Sprintf("Failed to list objects from S3: %v", err)})
		return
	}

	contents := []Content{}
	for _, obj := range s3objects {
		contents = append(contents, Content{*obj.Key, *obj.LastModified, *obj.Size, *obj.StorageClass})
	}
	commonPrefixes := []CommonPrefix{}
	for _, commonPrefix := range s3commonPrefixes {
		commonPrefixes = append(commonPrefixes, CommonPrefix{*commonPrefix.Prefix})
	}

	c.XML(200, ListBucketResult{bucket, prefix, delimiter, len(s3objects), contents, commonPrefixes})
}

func s3ListKeys(bucket string, prefix string, delimiter string) ([]string, error) {
	objects, _, err := s3ListObjects(bucket, prefix, delimiter)
	if err != nil {
		return nil, err
	}

	keys := []string{}
	for _, obj := range objects {
		keys = append(keys, *obj.Key)
	}

	return keys, nil
}

func s3ListObjects(bucket string, prefix string, delimiter string) ([]*s3.Object, []*s3.CommonPrefix, error) {
	s3objects := []*s3.Object{}
	s3CommonPrefixes := []*s3.CommonPrefix{}
	err := s3Client.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(delimiter),
	},
		func(page *s3.ListObjectsV2Output, lastPage bool) bool {
			s3objects = append(s3objects, page.Contents...)
			s3CommonPrefixes = append(s3CommonPrefixes, page.CommonPrefixes...)
			return !lastPage
		})

	if err != nil {
		return nil, nil, err
	}

	return s3objects, s3CommonPrefixes, nil
}

func s3Download(bucket string, key string, buf *aws.WriteAtBuffer) error {
	_, err := s3Downloader.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}
