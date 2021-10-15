// Copyright 2021 Adrian Chifor, Marshall Wace
// SPDX-FileCopyrightText: 2021 Marshall Wace <opensource@mwam.com>
// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/adrianchifor/go-parallel"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gin-gonic/gin"
	"github.com/mailgun/groupcache/v2"
	log "github.com/sirupsen/logrus"
)

var (
	peers        []string
	cacheGroup   *groupcache.Group
	cachePool    *groupcache.HTTPPool
	maxCacheSize int64
	ttl          int
	timeout      int
)

func initCachePool() {
	cachePool = groupcache.NewHTTPPoolOpts(fmt.Sprintf("http://%s:%d", host, port),
		&groupcache.HTTPPoolOptions{})

	if len(peers) > 0 {
		cachePool.Set(peers...)
	}

	cacheGroup = groupcache.NewGroup("s3", maxCacheSize<<20, groupcache.GetterFunc(cacheFiller))
}

func cacheFiller(ctx context.Context, cacheKey string, dest groupcache.Sink) error {
	log.Debugf("Pulling '%s' into cache from S3", cacheKey)
	keySplit := strings.Split(cacheKey, "#")
	bucket := keySplit[0]
	key := keySplit[1]
	buf := aws.NewWriteAtBuffer([]byte{})
	err := s3Download(bucket, key, buf)
	if err != nil {
		log.Errorf("Failed to download '%s' from S3: %v", cacheKey, err)
		return err
	}

	if ttl > 0 {
		log.Debugf("Pulled '%s' into buffer, adding to cache with %dm TTL", cacheKey, ttl)
		err = dest.SetBytes(buf.Bytes(), time.Now().Add(time.Minute*time.Duration(ttl)))
	} else {
		// "Disable" TTL - expire in 10 years
		log.Debugf("Pulled '%s' into buffer, adding to cache with 10 year TTL", cacheKey)
		err = dest.SetBytes(buf.Bytes(), time.Now().Add(time.Hour*87650))
	}
	if err != nil {
		log.Errorf("Failed to fill cache sink with '%s': %v", key, err)
		return err
	}

	log.Debugf("Pulled '%s' into cache", cacheKey)

	return nil
}

func restCacheGet(c *gin.Context) {
	bucket := strings.TrimSpace(c.Query("bucket"))
	if bucket == "" {
		c.JSON(400, gin.H{"error": "'bucket' not found in querystring parameters"})
		return
	}
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		c.JSON(400, gin.H{"error": "'key' not found in querystring parameters"})
		return
	}

	cacheKey := constructCacheKey(bucket, key)
	log.Debugf("Checking cache for '%s'", cacheKey)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(timeout))
	defer cancel()

	var cacheView groupcache.ByteView
	if err := cacheGroup.Get(ctx, cacheKey, groupcache.ByteViewSink(&cacheView)); err != nil {
		c.JSON(404, gin.H{"error": fmt.Sprintf("Blob '%s' not found", cacheKey)})
		return
	}

	extraHeaders := map[string]string{
		"Content-Disposition": fmt.Sprintf(`attachment; filename="%s"`, key),
	}
	log.Debugf("Sending '%s' bytes in response", cacheKey)
	c.DataFromReader(200, int64(cacheView.Len()), "application/octet-stream", cacheView.Reader(), extraHeaders)
}

func restCachePrewarm(c *gin.Context) {
	bucket := strings.TrimSpace(c.Query("bucket"))
	if bucket == "" {
		c.JSON(400, gin.H{"error": "'bucket' not found in querystring parameters"})
		return
	}
	prefix := strings.TrimSpace(c.Query("prefix"))
	if prefix == "" {
		c.JSON(400, gin.H{"error": "'prefix' not found in querystring parameters"})
		return
	}

	log.Debugf("Pre-warming cache with prefix '%s#%s'", bucket, prefix)
	keys, err := s3ListKeys(bucket, prefix, "")
	if err != nil {
		msg := fmt.Sprintf("Failed to list keys with prefix '%s' in S3 bucket '%s': %v", prefix, bucket, err)
		log.Errorf(msg)
		c.JSON(500, gin.H{"error": msg})
		return
	}
	if len(keys) == 0 {
		c.JSON(404, gin.H{"error": fmt.Sprintf("No keys found with prefix '%s' in S3 bucket '%s'", prefix, bucket)})
		return
	}

	go func() {
		getPool := parallel.SmallJobPool()
		defer getPool.Close()

		for _, key := range keys {
			key := key
			// Prewarm 10 keys at a time
			getPool.AddJob(func() {
				cacheKey := constructCacheKey(bucket, key)
				log.Debugf("Pre-warming cache for '%s'", cacheKey)
				ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(timeout))
				defer cancel()

				var tmpCacheView groupcache.ByteView
				if err := cacheGroup.Get(ctx, cacheKey, groupcache.ByteViewSink(&tmpCacheView)); err != nil {
					log.Errorf("Failed to pre-warm cache with key '%s': %v", cacheKey, err)
				}
			})
		}
	}()

	c.JSON(200, gin.H{
		"message": fmt.Sprintf("Pre-warming cache in the background with prefix '%s' from S3 bucket '%s'", prefix, bucket),
		"error":   "",
	})
}

func restCacheInvalidate(c *gin.Context) {
	bucket := strings.TrimSpace(c.Query("bucket"))
	if bucket == "" {
		c.JSON(400, gin.H{"error": "'bucket' not found in querystring parameters"})
		return
	}
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		c.JSON(400, gin.H{"error": "'key' not found in querystring parameters"})
		return
	}

	cacheInvalidate(bucket, key)
	c.JSON(200, gin.H{
		"message": fmt.Sprintf("'%s' invalidated from cache", constructCacheKey(bucket, key)),
		"error":   "",
	})
}

func cacheInvalidate(bucket string, key string) {
	cacheKey := constructCacheKey(bucket, key)
	cacheGroup.Remove(context.Background(), cacheKey)
	log.Debugf("'%s' invalidated from cache", cacheKey)
}
