// Copyright 2021 Adrian Chifor, Marshall Wace
// SPDX-FileCopyrightText: 2021 Marshall Wace <opensource@mwam.com>
// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/adrianchifor/go-parallel"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gin-gonic/gin"
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

func s3Upload(c *gin.Context) {
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
			cacheKey := constructCacheKey(bucket, fullKey)
			go cacheGroup.Remove(context.Background(), cacheKey)
			log.Debugf("'%s' invalidated from cache", cacheKey)
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

func s3Delete(c *gin.Context) {
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
		log.Debugf("Deleting key '%s#%s' from S3", bucket, key)
		_, err := s3Client.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			msg := fmt.Sprintf("Failed to delete '%s#%s' from S3: %v", bucket, key, err)
			log.Errorf(msg)
			c.JSON(500, gin.H{"error": msg})
			return
		}

		s3Client.WaitUntilObjectNotExists(&s3.HeadObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		msg := fmt.Sprintf("Deleted '%s#%s' from S3", bucket, key)
		log.Debugf(msg)

		// Invalidate deleted blob if in-memory
		cacheKey := constructCacheKey(bucket, key)
		go cacheGroup.Remove(context.Background(), cacheKey)
		log.Debugf("'%s' invalidated from cache", cacheKey)

		c.JSON(200, gin.H{
			"message": msg,
			"error":   "",
		})
	} else {
		log.Debugf("Deleting prefix '%s#%s' from S3", bucket, prefix)
		keysToDelete, _ := s3ListKeys(bucket, prefix)
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
				go cacheGroup.Remove(context.Background(), constructCacheKey(bucket, key))
			}
		}

		c.JSON(200, gin.H{
			"message": msg,
			"error":   "",
		})
	}
}

func s3Download(bucket string, key string, buf *aws.WriteAtBuffer) error {
	_, err := s3Downloader.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

func s3List(c *gin.Context) {
	bucket := strings.TrimSpace(c.Query("bucket"))
	if bucket == "" {
		c.JSON(400, gin.H{"error": "'bucket' not found in querystring parameters"})
		return
	}
	prefix := strings.TrimSpace(c.Query("prefix"))

	keys, err := s3ListKeys(bucket, prefix)
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

func s3ListKeys(bucket string, prefix string) ([]string, error) {
	keys := []string{}
	err := s3Client.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	},
		func(page *s3.ListObjectsV2Output, lastPage bool) bool {
			for _, obj := range page.Contents {
				keys = append(keys, *obj.Key)
			}
			return !lastPage
		})

	if err != nil {
		return nil, err
	}

	return keys, nil
}
