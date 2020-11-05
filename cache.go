package main

import (
	"context"
	"fmt"
	"strings"
	"time"

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

	log.Debugf("Pulled '%s' into buffer, adding to cache with TTL %dm", cacheKey, ttl)

	err = dest.SetBytes(buf.Bytes(), time.Now().Add(time.Minute*time.Duration(ttl)))
	if err != nil {
		log.Errorf("Failed to fill cache sink with '%s': %v", key, err)
		return err
	}

	log.Debugf("Pulled '%s' into cache", cacheKey)

	return nil
}

func cacheGet(c *gin.Context) {
	bucket := strings.TrimSpace(c.Query("bucket"))
	if bucket == "" {
		c.String(400, "'bucket' not found in querystring parameters")
		return
	}
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		c.String(400, "'key' not found in querystring parameters")
		return
	}
	cacheKey := constructCacheKey(bucket, key)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(timeout))
	defer cancel()

	log.Debugf("Checking cache for '%s'", cacheKey)

	var data []byte
	if err := cacheGroup.Get(ctx, cacheKey, groupcache.AllocatingByteSliceSink(&data)); err != nil {
		c.String(404, "Blob not found")
		return
	}

	log.Debugf("Sending '%s' bytes in response", cacheKey)

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, key))
	c.Data(200, "application/octet-stream", data)
}

func cacheInvalidate(c *gin.Context) {
	bucket := strings.TrimSpace(c.Query("bucket"))
	if bucket == "" {
		c.String(400, "'bucket' not found in querystring parameters")
		return
	}
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		c.String(400, "'key' not found in querystring parameters")
		return
	}
	cacheKey := constructCacheKey(bucket, key)
	cacheGroup.Remove(context.Background(), cacheKey)
	msg := fmt.Sprintf("'%s' invalidated from cache", cacheKey)
	log.Debugf(msg)
	c.String(200, msg)
}
