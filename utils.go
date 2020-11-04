package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

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
