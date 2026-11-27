// Package observability holds the Prometheus metrics behind GitaiHub's SLOs:
// the four golden signals of the HTTP API (traffic, errors, latency,
// saturation) and the freshness of the live feed.
package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Metrics is the set of GitaiHub metrics.
type Metrics struct {
	Registry        *prometheus.Registry
	Requests        *prometheus.CounterVec
	Latency         *prometheus.HistogramVec
	InFlight        prometheus.Gauge
	FeedLag         *prometheus.HistogramVec
	WatchErrors     *prometheus.CounterVec
	FeedSubscribers prometheus.GaugeFunc
}

// New registers every metric on a fresh registry.
func New(subscribers func() float64) *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{
		Registry: reg,
		Requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "gitaihub_http_requests_total",
			Help: "HTTP requests by route and status code (traffic and errors; availability SLO).",
		}, []string{"route", "method", "code"}),
		Latency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gitaihub_http_request_duration_seconds",
			Help:    "HTTP request latency by route (latency SLO: p95 of reads under 500 ms).",
			Buckets: []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		}, []string{"route", "method"}),
		InFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "gitaihub_http_requests_in_flight",
			Help: "Requests being served (saturation).",
		}),
		FeedLag: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gitaihub_feed_lag_seconds",
			Help:    "Delay between gitai recording a changeset and GitaiHub storing it (freshness SLO: under 5 s).",
			Buckets: []float64{.1, .25, .5, 1, 2, 5, 10, 30, 60},
		}, []string{"repository"}),
		WatchErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "gitaihub_changeset_watch_errors_total",
			Help: "Changeset watch streams that ended with an error.",
		}, []string{"repository"}),
	}
	m.FeedSubscribers = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "gitaihub_feed_subscribers",
		Help: "Browsers following the live feed.",
	}, subscribers)
	reg.MustRegister(m.Requests, m.Latency, m.InFlight, m.FeedLag, m.WatchErrors, m.FeedSubscribers,
		collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	return m
}

// ChangesetIngested implements activity.Metrics.
func (m *Metrics) ChangesetIngested(repo string, lag time.Duration) {
	if lag < 0 {
		lag = 0
	}
	m.FeedLag.WithLabelValues(repo).Observe(lag.Seconds())
}

// WatchError implements activity.Metrics.
func (m *Metrics) WatchError(repo string) { m.WatchErrors.WithLabelValues(repo).Inc() }
