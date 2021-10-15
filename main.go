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
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const version string = "0.15.0"

var (
	host                   string
	port                   int
	maxMultipartMemory     int64
	peersFlag              string
	disableHttpMetricsFlag bool
	s3TransparentAPI       bool
	verbose                bool
	versionFlag            bool
)

func init() {
	flag.StringVar(&host, "host", "localhost", "Host/IP to identify self in peers list")
	flag.IntVar(&port, "port", 8080, "Server port")
	flag.IntVar(&metricsPort, "metrics-port", 9095, "Prometheus metrics port")
	flag.StringVar(&s3Endpoint, "s3-endpoint", "", "Custom S3 endpoint URL (defaults to AWS)")
	flag.BoolVar(&s3TransparentAPI, "s3-transparent-api", false,
		"Enable transparent S3 API for usage from awscli or SDKs (default false)")
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
	flag.IntVar(&ttl, "ttl", 60, "Blob time-to-live in cache in minutes (0 to never expire)")
	flag.IntVar(&timeout, "timeout", 5000, "Get blob timeout in milliseconds")
	flag.StringVar(&peersFlag, "peers", "",
		"Peers (default '', e.g. 'http://peer1:8080,http://peer2:8080')")
	flag.BoolVar(&disableHttpMetricsFlag, "disable-http-metrics", false,
		"Disable HTTP metrics (req/s, latency) when expecting high path cardinality (default false)")
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

	if !disableHttpMetricsFlag {
		router.Use(httpMetricsMiddleware())
	}

	router.MaxMultipartMemory = maxMultipartMemory << 20
	router.POST("/upload", restS3Upload)
	router.DELETE("/delete", restS3Delete)
	router.GET("/list", restS3List)
	router.GET("/get", restCacheGet)
	router.POST("/prewarm", restCachePrewarm)
	router.POST("/invalidate", restCacheInvalidate)
	router.GET("/_groupcache/s3/*blob", gin.WrapF(cachePool.ServeHTTP))
	router.DELETE("/_groupcache/s3/*blob", gin.WrapF(cachePool.ServeHTTP))
	router.GET("/healthz", func(c *gin.Context) {
		c.String(200, fmt.Sprintf("Version: %s", version))
	})

	if s3TransparentAPI {
		router.GET("/", transparentS3ListBuckets)
		router.GET("/:bucket", transparentS3ListObjects)
		router.HEAD("/:bucket/*key", func(c *gin.Context) {
			// Dummy headers and 200 response to proceed to GET
			c.Header("Content-Length", "0")
			c.Header("Last-Modified", time.Now().Format("Mon, 2 Jan 2006 15:04:05 MST"))
			c.String(200, "")
		})
		router.GET("/:bucket/*key", transparentS3Get)
		router.PUT("/:bucket/*key", transparentS3Put)
		router.DELETE("/:bucket/*key", transparentS3Delete)
	}

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
