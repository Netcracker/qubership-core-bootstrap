package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type DefaultMetricsCollector struct {
	registry *prometheus.Registry

	// Rate limit metrics
	totalRateLimitsActive prometheus.Gauge
	totalViolatingUsers   prometheus.Gauge
	rateLimitChecksTotal  *prometheus.CounterVec
	rateLimitResetsTotal  prometheus.Counter

	// API metrics
	apiRequestsTotal   *prometheus.CounterVec
	apiRequestDuration *prometheus.HistogramVec

	// Redis metrics
	redisOperationsTotal   *prometheus.CounterVec
	redisOperationDuration *prometheus.HistogramVec

	// Config metrics
	configReloadsTotal      prometheus.Counter
	configReloadErrorsTotal prometheus.Counter
}

func NewDefaultMetricsCollector() *DefaultMetricsCollector {
	registry := prometheus.NewRegistry()

	// Rate limit metrics
	totalRateLimitsActive := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ratelimit_active_limits_total",
		Help: "Total number of active rate limits in Redis",
	})
	registry.MustRegister(totalRateLimitsActive)

	totalViolatingUsers := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ratelimit_violating_users_total",
		Help: "Total number of users exceeding rate limits",
	})
	registry.MustRegister(totalViolatingUsers)

	rateLimitChecksTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ratelimit_checks_total",
		Help: "Total number of rate limit checks",
	}, []string{"key", "result"})
	registry.MustRegister(rateLimitChecksTotal)

	rateLimitResetsTotal := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ratelimit_resets_total",
		Help: "Total number of rate limit resets",
	})
	registry.MustRegister(rateLimitResetsTotal)

	// API metrics
	apiRequestsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ratelimit_api_requests_total",
		Help: "Total number of API requests",
	}, []string{"endpoint", "method", "status"})
	registry.MustRegister(apiRequestsTotal)

	apiRequestDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ratelimit_api_request_duration_seconds",
		Help:    "API request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"endpoint", "method"})
	registry.MustRegister(apiRequestDuration)

	// Redis metrics
	redisOperationsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ratelimit_redis_operations_total",
		Help: "Total number of Redis operations",
	}, []string{"operation", "status"})
	registry.MustRegister(redisOperationsTotal)

	redisOperationDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ratelimit_redis_operation_duration_seconds",
		Help:    "Redis operation duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation"})
	registry.MustRegister(redisOperationDuration)

	// Config metrics
	configReloadsTotal := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ratelimit_config_reloads_total",
		Help: "Total number of config reloads",
	})
	registry.MustRegister(configReloadsTotal)

	configReloadErrorsTotal := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ratelimit_config_reload_errors_total",
		Help: "Total number of config reload errors",
	})
	registry.MustRegister(configReloadErrorsTotal)

	return &DefaultMetricsCollector{
		registry:                registry,
		totalRateLimitsActive:   totalRateLimitsActive,
		totalViolatingUsers:     totalViolatingUsers,
		rateLimitChecksTotal:    rateLimitChecksTotal,
		rateLimitResetsTotal:    rateLimitResetsTotal,
		apiRequestsTotal:        apiRequestsTotal,
		apiRequestDuration:      apiRequestDuration,
		redisOperationsTotal:    redisOperationsTotal,
		redisOperationDuration:  redisOperationDuration,
		configReloadsTotal:      configReloadsTotal,
		configReloadErrorsTotal: configReloadErrorsTotal,
	}
}

func (m *DefaultMetricsCollector) UpdateRateLimitMetrics(violatingCount int, activeLimitsCount int) {
	m.totalViolatingUsers.Set(float64(violatingCount))
	m.totalRateLimitsActive.Set(float64(activeLimitsCount))
}

func (m *DefaultMetricsCollector) RecordRateLimitCheck(key string, allowed bool, limit int) {
	result := "allowed"
	if !allowed {
		result = "rejected"
	}
	m.rateLimitChecksTotal.WithLabelValues(key, result).Inc()
}

func (m *DefaultMetricsCollector) RecordRateLimitReset(key string) {
	m.rateLimitResetsTotal.Inc()
}

func (m *DefaultMetricsCollector) RecordAPIRequest(endpoint, method, status string, duration float64) {
	m.apiRequestsTotal.WithLabelValues(endpoint, method, status).Inc()
	m.apiRequestDuration.WithLabelValues(endpoint, method).Observe(duration)
}

func (m *DefaultMetricsCollector) RecordRedisOperation(operation, status string, duration float64) {
	m.redisOperationsTotal.WithLabelValues(operation, status).Inc()
	m.redisOperationDuration.WithLabelValues(operation).Observe(duration)
}

func (m *DefaultMetricsCollector) RecordConfigReload(success bool) {
	m.configReloadsTotal.Inc()
	if !success {
		m.configReloadErrorsTotal.Inc()
	}
}

func (m *DefaultMetricsCollector) GetRegistry() *prometheus.Registry {
	return m.registry
}

// RecordRateLimit implements ratelimit.MetricsRecorder interface
func (m *DefaultMetricsCollector) RecordRateLimit(key string, allowed bool, current int, limit int) {
	result := "allowed"
	if !allowed {
		result = "rejected"
	}
	m.rateLimitChecksTotal.WithLabelValues(key, result).Inc()
}
