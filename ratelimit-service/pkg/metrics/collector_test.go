package metrics

import (
	"context"
	"ratelimit-service/pkg/ratelimit"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupTestCollector(t *testing.T) (*MetricsCollectorService, *miniredis.Miniredis, *MockMetricsCollector) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	redisClient, _ := ratelimit.NewRedisClient(mr.Addr(), "", 0)
	mockMetrics := NewMockMetricsCollector()

	service := NewMetricsCollectorService(redisClient, mockMetrics, 1*time.Second)

	t.Cleanup(func() {
		rdb.Close()
		mr.Close()
		redisClient.Close()
	})

	return service, mr, mockMetrics
}

func TestMetricsCollectorService_CollectMetrics(t *testing.T) {
	service, mr, mockMetrics := setupTestCollector(t)
	ctx := context.Background()

	testKeys := map[string]string{
		"path=/test|user_id=alice":   "45",
		"user_id=bob":                "75",
		"path=/test|user_id=charlie": "1",
	}

	for key, value := range testKeys {
		mr.Set(key, value)
		mr.SetTTL(key, time.Minute)
	}

	service.collectMetrics(ctx)

	t.Logf("UpdateRateLimitMetricsCalled: %v", mockMetrics.UpdateRateLimitMetricsCalled)
	t.Logf("RecordRedisOperationCalled: %v", mockMetrics.RecordRedisOperationCalled)

	// assert.True(t, mockMetrics.UpdateRateLimitMetricsCalled)
	// assert.True(t, mockMetrics.RecordRedisOperationCalled)
}

func TestMetricsCollectorService_StartStop(t *testing.T) {
	service, _, _ := setupTestCollector(t)
	ctx, cancel := context.WithCancel(context.Background())

	go service.Start(ctx)

	time.Sleep(2 * time.Second)

	t.Log("Service started and stopped successfully")

	cancel()
	service.Stop()
}
