package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/webpage-analyser-server/internal/constants"
)

// Metrics holds all Prometheus metrics for the application
type Metrics struct {
	RequestDuration   *prometheus.HistogramVec
	CacheHits        prometheus.Counter
	CacheMisses      prometheus.Counter
	LinkCheckDuration prometheus.Histogram
}


func New() *Metrics {
	m := &Metrics{
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    constants.MetricRequestDurationName,
				Help:    constants.MetricRequestDurationHelp,
				Buckets: prometheus.DefBuckets,
			},
			[]string{"status"},
		),
		CacheHits: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: constants.MetricCacheHitsName,
				Help: constants.MetricCacheHitsHelp,
			},
		),
		CacheMisses: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: constants.MetricCacheMissesName,
				Help: constants.MetricCacheMissesHelp,
			},
		),
		LinkCheckDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    constants.MetricLinkCheckDurationName,
				Help:    constants.MetricLinkCheckDurationHelp,
				Buckets: prometheus.DefBuckets,
			},
		),
	}

	// Register all metrics
	prometheus.MustRegister(m.RequestDuration)
	prometheus.MustRegister(m.CacheHits)
	prometheus.MustRegister(m.CacheMisses)
	prometheus.MustRegister(m.LinkCheckDuration)

	return m
} 