package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	devices  prometheus.Gauge
	info     *prometheus.GaugeVec
	duration *prometheus.HistogramVec
}

func NewMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		devices: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "samplebox",
			Name:      "connected_devices",
			Help:      "Number of currently connected devices.",
		}),
		info: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "samplebox",
			Name:      "info",
			Help:      "Information about the My App environment.",
		},
			[]string{"version"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "samplebox",
			Name:      "request_duration_seconds",
			Help:      "Duration of the request.",
			// 4 times larger for apdex score
			// Buckets: prometheus.ExponentialBuckets(0.1, 1.5, 5),
			// Buckets: prometheus.LinearBuckets(0.1, 5, 5),
			Buckets: []float64{0.1, 0.15, 0.2, 0.25, 0.3},
		}, []string{"status", "method"}),
	}
	reg.MustRegister(m.devices, m.info)
	return m
}
