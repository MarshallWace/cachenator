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

const version string = "0.3.0"

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
	flag.StringVar(&s3Endpoint, "s3-endpoint", "", "Custom S3 endpoint URL (defaults to AWS)")
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
	runServer()
}

func checkFlags() {
	if verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if versionFlag {
		log.Infof("Falcon version %s", version)
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

	listenAddr := fmt.Sprintf("127.0.0.1:%d", port)
	if os.Getenv("GIN_MODE") == "release" {
		listenAddr = fmt.Sprintf("0.0.0.0:%d", port)
	}

	router := gin.Default()
	router.MaxMultipartMemory = maxMultipartMemory << 20
	router.POST("/upload", s3Upload)
	router.GET("/get", cacheGet)
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

	go serverGracefulShutdown(server, quit, done)

	log.Infof("HTTP server is ready to handle requests at %s", listenAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server could not listen on %s: %v\n", listenAddr, err)
	}

	<-done
	log.Info("HTTP server stopped")
}
