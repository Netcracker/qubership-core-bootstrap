package metrics

import "github.com/prometheus/client_golang/prometheus"

type MetricsCollector interface {
	// Rate limit metrics
	UpdateRateLimitMetrics(violatingCount int, activeLimitsCount int)
	RecordRateLimitCheck(key string, allowed bool, limit int)
	RecordRateLimitReset(key string)

	// API metrics
	RecordAPIRequest(endpoint, method, status string, duration float64)

	// Redis metrics
	RecordRedisOperation(operation, status string, duration float64)

	// Config metrics
	RecordConfigReload(success bool)

	GetRegistry() *prometheus.Registry

	// RecordRateLimit implements ratelimit.MetricsRecorder
	RecordRateLimit(key string, allowed bool, current int, limit int)
}
