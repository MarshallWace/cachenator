package main

import (
	"bytes"
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
	peers      []string
	cacheGroup *groupcache.Group
	cachePool  *groupcache.HTTPPool
	ttl        int
	timeout    int
)

func initCachePool() {
	cachePool = groupcache.NewHTTPPoolOpts(fmt.Sprintf("http://%s:%d", host, port),
		&groupcache.HTTPPoolOptions{})

	if len(peers) > 0 {
		cachePool.Set(peers...)
	}

	cacheGroup = groupcache.NewGroup("s3", maxBlobSize<<20, groupcache.GetterFunc(cacheFiller))
}

func cacheFiller(ctx context.Context, key string, dest groupcache.Sink) error {
	log.Debugf("Pulling '%s' blob into cache from S3", key)
	buf := aws.NewWriteAtBuffer([]byte{})
	err := s3Download(key, buf)
	if err != nil {
		log.Errorf("Failed to download blob '%s' from S3: %v", key, err)
		return err
	}

	err = dest.SetBytes(buf.Bytes(), time.Now().Add(time.Minute*time.Duration(ttl)))
	if err != nil {
		log.Errorf("Failed to fill cache sink with blob '%s': %v", key, err)
		return err
	}

	return nil
}

func cacheGet(c *gin.Context) {
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		c.JSON(400, gin.H{
			"error": "'key' not found in querystring parameters",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(timeout))
	defer cancel()

	var data []byte
	if err := cacheGroup.Get(ctx, key, groupcache.AllocatingByteSliceSink(&data)); err != nil {
		c.String(404, "Blob not found")
		return
	}

	reader := bytes.NewReader(data)
	extraHeaders := map[string]string{
		"Content-Disposition": fmt.Sprintf(`attachment; filename="%s"`, key),
	}
	c.DataFromReader(200, int64(len(data)), "application/octet-stream", reader, extraHeaders)

}
