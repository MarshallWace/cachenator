package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mailgun/groupcache/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	metricsPort                       int
	groupGetsMetric                   prometheus.Gauge
	groupCacheHitsMetric              prometheus.Gauge
	groupPeersGetHighestLatencyMetric prometheus.Gauge
	groupPeerLoadsMetric              prometheus.Gauge
	groupPeerErrorsMetric             prometheus.Gauge
	groupLoadsMetric                  prometheus.Gauge
	groupLoadsDedupedMetric           prometheus.Gauge
	groupLocalLoadsMetric             prometheus.Gauge
	groupLocalLoadErrsMetric          prometheus.Gauge
	groupServerRequestsMetric         prometheus.Gauge
	cacheBytesMetric                  *prometheus.GaugeVec
	cacheItemsMetric                  *prometheus.GaugeVec
	cacheGetsMetric                   *prometheus.GaugeVec
	cacheHitsMetric                   *prometheus.GaugeVec
	cacheEvictionsMetric              *prometheus.GaugeVec
)

func initMetrics() {
	groupGetsMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cachenator_gets_total",
		Help: "Total number of get requests",
	})
	groupCacheHitsMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cachenator_hits_total",
		Help: "Total number of both main and hot cache hits",
	})
	groupPeersGetHighestLatencyMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cachenator_peers_get_highest_latency",
		Help: "Highest latency get request from peers",
	})
	groupPeerLoadsMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cachenator_peer_loads_total",
		Help: "Total number of remote loads or remote cache hits",
	})
	groupPeerErrorsMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cachenator_peer_errors_total",
		Help: "Total number of peer errors",
	})
	groupLoadsMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cachenator_loads_total",
		Help: "Total number of both local and remote cache loads",
	})
	groupLoadsDedupedMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cachenator_loads_deduped_total",
		Help: "Total number of deduplicated cache loads",
	})
	groupLocalLoadsMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cachenator_local_loads_total",
		Help: "Total number of local cache loads",
	})
	groupLocalLoadErrsMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cachenator_local_load_errors_total",
		Help: "Total number of local cache load errors",
	})
	groupServerRequestsMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cachenator_server_requests_total",
		Help: "Total number of gets from other peers",
	})
	cacheBytesMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cachenator_cache_bytes",
		Help: "Current (main/hot) cache bytes",
	}, []string{"type"})
	cacheItemsMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cachenator_cache_items",
		Help: "Current (main/hot) cache items",
	}, []string{"type"})
	cacheGetsMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cachenator_cache_gets_total",
		Help: "Total number of (main/hot) cache get requests",
	}, []string{"type"})
	cacheHitsMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cachenator_cache_hits_total",
		Help: "Total number of (main/hot) cache hits",
	}, []string{"type"})
	cacheEvictionsMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cachenator_cache_evictions_total",
		Help: "Total number of (main/hot) cache evictions",
	}, []string{"type"})
}

func collectMetrics() {
	cacheTypes := []groupcache.CacheType{groupcache.MainCache, groupcache.HotCache}

	for {
		groupGetsMetric.Set(float64(cacheGroup.Stats.Gets.Get()))
		groupCacheHitsMetric.Set(float64(cacheGroup.Stats.CacheHits.Get()))
		groupPeersGetHighestLatencyMetric.Set(
			float64(cacheGroup.Stats.GetFromPeersLatencyLower.Get()))
		groupPeerLoadsMetric.Set(float64(cacheGroup.Stats.PeerLoads.Get()))
		groupPeerErrorsMetric.Set(float64(cacheGroup.Stats.PeerErrors.Get()))
		groupLoadsMetric.Set(float64(cacheGroup.Stats.Loads.Get()))
		groupLoadsDedupedMetric.Set(float64(cacheGroup.Stats.LoadsDeduped.Get()))
		groupLocalLoadsMetric.Set(float64(cacheGroup.Stats.LocalLoads.Get()))
		groupLocalLoadErrsMetric.Set(float64(cacheGroup.Stats.LocalLoadErrs.Get()))
		groupServerRequestsMetric.Set(float64(cacheGroup.Stats.ServerRequests.Get()))

		for _, cacheType := range cacheTypes {
			cacheBytesMetric.WithLabelValues(cacheTypeName(cacheType)).Set(
				float64(cacheGroup.CacheStats(cacheType).Bytes))
			cacheItemsMetric.WithLabelValues(cacheTypeName(cacheType)).Set(
				float64(cacheGroup.CacheStats(cacheType).Items))
			cacheGetsMetric.WithLabelValues(cacheTypeName(cacheType)).Set(
				float64(cacheGroup.CacheStats(cacheType).Gets))
			cacheHitsMetric.WithLabelValues(cacheTypeName(cacheType)).Set(
				float64(cacheGroup.CacheStats(cacheType).Hits))
			cacheEvictionsMetric.WithLabelValues(cacheTypeName(cacheType)).Set(
				float64(cacheGroup.CacheStats(cacheType).Evictions))
		}

		time.Sleep(10 * time.Second)
	}
}

func cacheTypeName(cacheType groupcache.CacheType) string {
	if cacheType == groupcache.MainCache {
		return "main"
	}
	return "hot"
}

func runMetricsServer() {
	http.Handle("/metrics", promhttp.Handler())

	addr := fmt.Sprintf("127.0.0.1:%d", metricsPort)
	if os.Getenv("GIN_MODE") == "release" {
		addr = fmt.Sprintf("0.0.0.0:%d", metricsPort)
	}

	log.Infof("Prometheus metrics HTTP server listening at %s/metrics", addr)
	http.ListenAndServe(addr, nil)
}
