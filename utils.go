// SPDX-FileCopyrightText: 2022 Marshall Wace <opensource@mwam.com>
// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func cleanupPeers(peers []string) []string {
	cleanedPeers := []string{}
	for _, peer := range peers {
		cleanPeer := strings.TrimSpace(peer)
		if !strings.Contains(cleanPeer, "http://") {
			cleanPeer = fmt.Sprintf("http://%s", cleanPeer)
		}
		if strings.Count(cleanPeer, ":") < 2 {
			cleanPeer = fmt.Sprintf("%s:%d", cleanPeer, port)
		}
		cleanedPeers = append(cleanedPeers, cleanPeer)
	}
	return cleanedPeers
}

func serverGracefulShutdown(server *http.Server, quit <-chan os.Signal, done chan<- bool) {
	<-quit
	log.Info("HTTP server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	server.SetKeepAlivesEnabled(false)
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Could not gracefully shutdown the HTTP server: %v\n", err)
	}
	close(done)
}

func constructCacheKey(bucket string, key string) string {
	return fmt.Sprintf("%s#%s", bucket, key)
}

func jsonLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := getDurationInMillseconds(start)

		entry := log.WithFields(log.Fields{
			"client_ip": getClientIP(c),
			"duration":  duration,
			"method":    c.Request.Method,
			"path":      c.Request.RequestURI,
			"status":    c.Writer.Status(),
			"referrer":  c.Request.Referer(),
		})

		if c.Writer.Status() >= 500 {
			entry.Error(c.Errors.String())
		} else {
			entry.Info("")
		}
	}
}

func httpMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.RequestURI, "/_groupcache") {
			return
		}

		start := time.Now()
		c.Next()
		duration := getDurationInMillseconds(start)

		httpRequestsMetric.WithLabelValues(
			c.Request.Method,
			c.Request.RequestURI,
			strconv.Itoa(c.Writer.Status()),
		).Inc()

		httpRequestsLatencyMetric.WithLabelValues(
			c.Request.Method,
			c.Request.RequestURI,
			strconv.Itoa(c.Writer.Status()),
		).Set(duration)
	}
}

func getDurationInMillseconds(start time.Time) float64 {
	end := time.Now()
	duration := end.Sub(start)
	milliseconds := float64(duration) / float64(time.Millisecond)
	rounded := float64(int(milliseconds*100+.5)) / 100
	return rounded
}

func getClientIP(c *gin.Context) string {
	requester := c.Request.Header.Get("X-Forwarded-For")
	if len(requester) == 0 {
		requester = c.Request.Header.Get("X-Real-IP")
	}
	if len(requester) == 0 {
		requester = c.Request.RemoteAddr
	}
	if strings.Contains(requester, ",") {
		requester = strings.Split(requester, ",")[0]
	}

	return requester
}

func unsupportedRequest(c *gin.Context) {
	c.String(400, "Unsupported request under read-only mode.")
}
