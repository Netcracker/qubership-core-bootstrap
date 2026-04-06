package metrics

import (
	"context"
	"time"

	"ratelimit-service/pkg/ratelimit"

	"k8s.io/klog/v2"
)

type MetricsCollectorService struct {
	redisClient *ratelimit.RedisClient
	metrics     MetricsCollector
	interval    time.Duration
	stopCh      chan struct{}
}

func NewMetricsCollectorService(redisClient *ratelimit.RedisClient, metrics MetricsCollector, interval time.Duration) *MetricsCollectorService {
	if interval == 0 {
		interval = 30 * time.Second
	}

	return &MetricsCollectorService{
		redisClient: redisClient,
		metrics:     metrics,
		interval:    interval,
		stopCh:      make(chan struct{}),
	}
}

func (s *MetricsCollectorService) Start(ctx context.Context) {
	klog.Info("Starting metrics collector service")

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.collectMetrics(ctx)

	for {
		select {
		case <-ctx.Done():
			klog.Info("Stopping metrics collector service")
			return
		case <-s.stopCh:
			klog.Info("Metrics collector service stopped")
			return
		case <-ticker.C:
			s.collectMetrics(ctx)
		}
	}
}

func (s *MetricsCollectorService) Stop() {
	close(s.stopCh)
}

func (s *MetricsCollectorService) collectMetrics(ctx context.Context) {
	start := time.Now()

	violating, err := s.redisClient.GetViolatingUsers(ctx)
	if err != nil {
		klog.Errorf("Failed to get violating users: %v", err)
		s.metrics.RecordConfigReload(false)
		return
	}

	stats, err := s.redisClient.GetAllStatistics(ctx)
	if err != nil {
		klog.Errorf("Failed to get statistics: %v", err)
		s.metrics.RecordConfigReload(false)
		return
	}

	s.metrics.UpdateRateLimitMetrics(len(violating), stats.TotalKeys)
	s.metrics.RecordRedisOperation("collect", "success", time.Since(start).Seconds())

	klog.V(4).Infof("Metrics updated: %d violating users, %d active limits",
		len(violating), stats.TotalKeys)
}
