package metrics

import (
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds Prometheus metric collectors
type Metrics struct {
	// gRPC metrics
	grpcRequestsTotal   *prometheus.CounterVec
	grpcRequestDuration *prometheus.HistogramVec
	grpcRequestsInFlight *prometheus.GaugeVec

	// HTTP metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestsInFlight *prometheus.GaugeVec

	registry *prometheus.Registry
}

// Config holds configuration for metrics
type Config struct {
	Namespace string
	Subsystem string
}

// sanitizeName replaces invalid characters in metric names with underscores
// Prometheus metric names must match [a-zA-Z_:][a-zA-Z0-9_:]*
func sanitizeName(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i == 0 {
			// First character must be letter or underscore
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' {
				result.WriteRune(r)
			} else {
				result.WriteRune('_')
			}
		} else {
			// Subsequent characters can be letter, digit, or underscore
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == ':' {
				result.WriteRune(r)
			} else {
				result.WriteRune('_')
			}
		}
	}
	return result.String()
}

// New creates a new Metrics instance with the given configuration
func New(cfg Config) *Metrics {
	if cfg.Namespace == "" {
		cfg.Namespace = "app"
	}

	// Sanitize namespace and subsystem for Prometheus naming requirements
	cfg.Namespace = sanitizeName(cfg.Namespace)
	cfg.Subsystem = sanitizeName(cfg.Subsystem)

	m := &Metrics{
		registry: prometheus.NewRegistry(),
	}

	// gRPC metrics
	m.grpcRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "grpc_requests_total",
			Help:      "Total number of gRPC requests",
		},
		[]string{"method", "code"},
	)

	m.grpcRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "grpc_request_duration_seconds",
			Help:      "Duration of gRPC requests in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	m.grpcRequestsInFlight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "grpc_requests_in_flight",
			Help:      "Number of gRPC requests currently being processed",
		},
		[]string{"method"},
	)

	// HTTP metrics
	m.httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "code"},
	)

	m.httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "http_request_duration_seconds",
			Help:      "Duration of HTTP requests in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	m.httpRequestsInFlight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "http_requests_in_flight",
			Help:      "Number of HTTP requests currently being processed",
		},
		[]string{"method", "path"},
	)

	// Register all metrics
	m.registry.MustRegister(
		m.grpcRequestsTotal,
		m.grpcRequestDuration,
		m.grpcRequestsInFlight,
		m.httpRequestsTotal,
		m.httpRequestDuration,
		m.httpRequestsInFlight,
	)

	return m
}

// Handler returns an http.Handler that serves Prometheus metrics
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Register registers the metrics with the default prometheus registry
func (m *Metrics) Register() {
	prometheus.MustRegister(
		m.grpcRequestsTotal,
		m.grpcRequestDuration,
		m.grpcRequestsInFlight,
		m.httpRequestsTotal,
		m.httpRequestDuration,
		m.httpRequestsInFlight,
	)
}
