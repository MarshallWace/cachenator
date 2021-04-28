// Copyright 2021 Adrian Chifor, Marshall Wace
// SPDX-FileCopyrightText: 2021 Marshall Wace <opensource@mwam.com>
// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const version string = "0.13.1"

var (
	host               string
	port               int
	maxMultipartMemory int64
	peersFlag          string
	verbose            bool
	versionFlag        bool
)

func init() {
	flag.StringVar(&host, "host", "localhost", "Host/IP to identify self in peers list")
	flag.IntVar(&port, "port", 8080, "Server port")
	flag.IntVar(&metricsPort, "metrics-port", 9095, "Prometheus metrics port")
	flag.StringVar(&s3Endpoint, "s3-endpoint", "", "Custom S3 endpoint URL (defaults to AWS)")
	flag.BoolVar(&s3ForcePathStyle, "s3-force-path-style", false,
		"Force S3 path bucket addressing (endpoint/bucket/key vs. bucket.endpoint/key) (default false)")
	flag.Int64Var(&uploadPartSize, "s3-upload-part-size", 5,
		"Buffer size in megabytes when uploading blob chunks to S3 (minimum 5)")
	flag.IntVar(&uploadConcurrency, "s3-upload-concurrency", 10,
		"Number of goroutines to spin up when uploading blob chunks to S3")
	flag.Int64Var(&downloadPartSize, "s3-download-part-size", 5,
		"Size in megabytes to request from S3 for each blob chunk (minimum 5)")
	flag.IntVar(&downloadConcurrency, "s3-download-concurrency", 10,
		"Number of goroutines to spin up when downloading blob chunks from S3")
	flag.Int64Var(&maxMultipartMemory, "max-multipart-memory", 128,
		"Max memory in megabytes for /upload multipart form parsing")
	flag.Int64Var(&maxCacheSize, "max-cache-size", 512,
		"Max cache size in megabytes. If size goes above, oldest keys will be evicted")
	flag.IntVar(&ttl, "ttl", 60, "Blob time-to-live in cache in minutes")
	flag.IntVar(&timeout, "timeout", 5000, "Get blob timeout in milliseconds")
	flag.StringVar(&peersFlag, "peers", "",
		"Peers (default '', e.g. 'http://peer1:8080,http://peer2:8080')")
	flag.BoolVar(&verbose, "verbose", false, "Verbose logs")
	flag.BoolVar(&versionFlag, "version", false, "Version")
	flag.Parse()
}

func main() {
	checkFlags()
	initS3()
	initCachePool()
	initMetrics()
	go collectMetrics()
	runServer()
}

func checkFlags() {
	if verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if versionFlag {
		log.Infof("Cachenator version %s", version)
		os.Exit(0)
	}

	peers = []string{}
	if peersFlag != "" {
		peers = strings.Split(peersFlag, ",")
		peers = cleanupPeers(peers)
	}
}

func runServer() {
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	router := gin.Default()

	listenAddr := fmt.Sprintf("127.0.0.1:%d", port)
	if os.Getenv("GIN_MODE") == "release" {
		listenAddr = fmt.Sprintf("0.0.0.0:%d", port)
		log.SetFormatter(&log.JSONFormatter{})
		router.Use(jsonLogMiddleware())
	} else {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp: true,
		})
	}

	router.Use(httpMetricsMiddleware())
	router.MaxMultipartMemory = maxMultipartMemory << 20
	router.POST("/upload", s3Upload)
	router.DELETE("/delete", s3Delete)
	router.GET("/list", s3List)
	router.GET("/get", cacheGet)
	router.POST("/prewarm", cachePrewarm)
	router.POST("/invalidate", cacheInvalidate)
	router.GET("/_groupcache/s3/*blob", gin.WrapF(cachePool.ServeHTTP))
	router.DELETE("/_groupcache/s3/*blob", gin.WrapF(cachePool.ServeHTTP))
	router.GET("/healthz", func(c *gin.Context) {
		c.String(200, "UP")
	})

	server := &http.Server{
		Addr:    listenAddr,
		Handler: router,
	}

	fmt.Println(`
		┌────────────────────────────────────────┐
		│░█▀▀░█▀█░█▀▀░█░█░█▀▀░█▀█░█▀█░▀█▀░█▀█░█▀▄│
		│░█░░░█▀█░█░░░█▀█░█▀▀░█░█░█▀█░░█░░█░█░█▀▄│
		│░▀▀▀░▀░▀░▀▀▀░▀░▀░▀▀▀░▀░▀░▀░▀░░▀░░▀▀▀░▀░▀│
		└────────────────────────────────────────┘
	`)
	log.Infof("Running (v%s): %s", version, strings.Join(os.Args, " "))

	go runMetricsServer()
	go serverGracefulShutdown(server, quit, done)

	log.Infof("HTTP server is ready to handle requests at %s", listenAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server could not listen on %s: %v\n", listenAddr, err)
	}

	<-done
	log.Info("HTTP server stopped")
}
