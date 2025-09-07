package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var (
	// HTTP request metrics
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)
	httpRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

var initMetricsOnce sync.Once
var registry *prometheus.Registry

// InitMetrics initializes and registers Prometheus metrics.
func InitMetrics() *prometheus.Registry {
	initMetricsOnce.Do(func() {
		registry = prometheus.NewRegistry()

		// Register HTTP metrics
		registry.MustRegister(httpRequestsTotal)
		registry.MustRegister(httpRequestDurationSeconds)

		// Register Go runtime metrics
		registry.MustRegister(collectors.NewGoCollector())
		registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

		log.Info().Msg("Prometheus metrics initialized.")
	})
	return registry
}

// MetricsHandler returns an http.Handler that serves Prometheus metrics.
func MetricsHandler(reg *prometheus.Registry) http.Handler {
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}

// HTTPMetricsMiddleware collects HTTP request metrics.
func HTTPMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// Use a custom ResponseWriter to capture status code
		lw := &loggingResponseWriter{w, http.StatusOK}
		next.ServeHTTP(lw, r)

		duration := time.Since(start).Seconds()
		method := r.Method
		path := r.URL.Path
		status := strconv.Itoa(lw.statusCode)

		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDurationSeconds.WithLabelValues(method, path).Observe(duration)
	})
}

// loggingResponseWriter is a wrapper to capture the HTTP status code.
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// GetMetricsInfo returns current metrics information as a map
func GetMetricsInfo() map[string]interface{} {
	if registry == nil {
		return map[string]interface{}{
			"status": "metrics not initialized",
		}
	}

	metricsInfo := make(map[string]interface{})

	// Gather metrics from the registry
	metricFamilies, err := registry.Gather()
	if err != nil {
		log.Error().Err(err).Msg("failed to gather metrics")
		return map[string]interface{}{
			"status": "error gathering metrics",
			"error":  err.Error(),
		}
	}

	// Process HTTP request metrics
	httpMetrics := make(map[string]interface{})
	totalRequests := 0.0

	// Process Go runtime metrics
	runtimeMetrics := make(map[string]interface{})

	// Process each metric family
	for _, mf := range metricFamilies {
		metricName := mf.GetName()

		switch {
		case strings.HasPrefix(metricName, "http_requests_total"):
			// Sum up all HTTP requests
			for _, metric := range mf.GetMetric() {
				if metric.Counter != nil {
					totalRequests += metric.Counter.GetValue()
				}
			}
			httpMetrics["total_requests"] = totalRequests

		case strings.HasPrefix(metricName, "go_goroutines"):
			if len(mf.GetMetric()) > 0 && mf.GetMetric()[0].Gauge != nil {
				runtimeMetrics["goroutines"] = int(mf.GetMetric()[0].Gauge.GetValue())
			}

		case strings.HasPrefix(metricName, "go_memstats_alloc_bytes"):
			if len(mf.GetMetric()) > 0 && mf.GetMetric()[0].Gauge != nil {
				runtimeMetrics["allocated_bytes"] = int64(mf.GetMetric()[0].Gauge.GetValue())
			}

		case strings.HasPrefix(metricName, "go_memstats_sys_bytes"):
			if len(mf.GetMetric()) > 0 && mf.GetMetric()[0].Gauge != nil {
				runtimeMetrics["system_bytes"] = int64(mf.GetMetric()[0].Gauge.GetValue())
			}

		case strings.HasPrefix(metricName, "process_resident_memory_bytes"):
			if len(mf.GetMetric()) > 0 && mf.GetMetric()[0].Gauge != nil {
				runtimeMetrics["resident_memory_bytes"] = int64(mf.GetMetric()[0].Gauge.GetValue())
			}
		}
	}

	metricsInfo["http"] = httpMetrics
	metricsInfo["runtime"] = runtimeMetrics
	metricsInfo["total_metrics_collected"] = len(metricFamilies)

	return metricsInfo
}
