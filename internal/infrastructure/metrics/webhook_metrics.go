package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// WebhookMetrics holds simplified worker processing metrics
type WebhookMetrics struct {
	// Histogram for total worker processing duration by status code and retry level
	workerProcessingDuration prometheus.HistogramVec

	// Counter for total queue items processed by workers by status code and retry level
	workerProcessingTotal prometheus.CounterVec
}

// NewWebhookMetrics creates and registers simplified worker processing metrics
func NewWebhookMetrics() *WebhookMetrics {
	return &WebhookMetrics{
		// Worker processing duration by status code and retry level
		workerProcessingDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "worker_processing_duration_seconds",
				Help:    "Total time for worker to process one queue item by status code and retry level",
				Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60}, // seconds
			},
			[]string{"status_code", "retry_level"},
		),

		// Worker processing count by status code and retry level
		workerProcessingTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "worker_processing_total",
				Help: "Total number of queue items processed by workers by status code and retry level",
			},
			[]string{"status_code", "retry_level"},
		),
	}
}

// RecordWorkerProcessing records worker processing metrics by status code and retry level
func (m *WebhookMetrics) RecordWorkerProcessing(statusCode int, retryLevel int, duration time.Duration) {
	statusCodeStr := strconv.Itoa(statusCode)
	retryLevelStr := strconv.Itoa(retryLevel)

	// Record processing duration by status code and retry level
	m.workerProcessingDuration.WithLabelValues(statusCodeStr, retryLevelStr).Observe(duration.Seconds())

	// Record processing count by status code and retry level
	m.workerProcessingTotal.WithLabelValues(statusCodeStr, retryLevelStr).Inc()
}
