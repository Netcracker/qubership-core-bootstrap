package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestLimiter(t *testing.T) (*Limiter, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	limiter := NewLimiter(rdb, nil)
	return limiter, mr
}

func TestFixedWindowLimiter(t *testing.T) {
	limiter, _ := setupTestLimiter(t)
	ctx := context.Background()
	key := "test:user:alice"
	limit := 10
	window := time.Second

	for i := 0; i < limit; i++ {
		result, err := limiter.AllowWithAlgorithm(ctx, key, limit, window, AlgorithmFixedWindow)
		require.NoError(t, err)
		assert.True(t, result.Allowed, "Request %d should be allowed", i+1)
		t.Logf("Request %d: allowed=%v, remaining=%d", i+1, result.Allowed, result.LimitRemaining)
	}

	result, err := limiter.AllowWithAlgorithm(ctx, key, limit, window, AlgorithmFixedWindow)
	require.NoError(t, err)
	t.Logf("Request 11: allowed=%v, remaining=%d", result.Allowed, result.LimitRemaining)
	assert.False(t, result.Allowed, "11th request should be rejected")
}

func TestSlidingWindowLimiter(t *testing.T) {
	limiter, _ := setupTestLimiter(t)
	ctx := context.Background()
	key := "test:user:bob"
	limit := 5
	window := 1 * time.Second

	results := make([]*Result, 15)
	for i := 0; i < 15; i++ {
		result, err := limiter.AllowWithAlgorithm(ctx, key, limit, window, AlgorithmSlidingWindow)
		require.NoError(t, err)
		results[i] = result
		time.Sleep(50 * time.Millisecond)
	}

	allowedCount := 0
	rejectedCount := 0
	for _, r := range results {
		if r.Allowed {
			allowedCount++
		} else {
			rejectedCount++
		}
	}

	t.Logf("Limit: %d, Window: %v", limit, window)
	t.Logf("Allowed: %d, Rejected: %d", allowedCount, rejectedCount)

	assert.Greater(t, allowedCount, 0, "Some requests should be allowed")
	assert.Greater(t, rejectedCount, 0, "Some requests should be rejected")
}

func TestTokenBucketLimiter(t *testing.T) {
	limiter, _ := setupTestLimiter(t)
	ctx := context.Background()
	key := "test:user:charlie"
	limit := 10
	window := time.Second

	for i := 0; i < limit; i++ {
		result, err := limiter.AllowWithAlgorithm(ctx, key, limit, window, AlgorithmTokenBucket)
		require.NoError(t, err)
		assert.True(t, result.Allowed, "Request %d should be allowed", i+1)
		t.Logf("Request %d: allowed=%v, remaining=%d", i+1, result.Allowed, result.LimitRemaining)
	}

	result, err := limiter.AllowWithAlgorithm(ctx, key, limit, window, AlgorithmTokenBucket)
	require.NoError(t, err)
	t.Logf("Request 11: allowed=%v, remaining=%d", result.Allowed, result.LimitRemaining)
	assert.False(t, result.Allowed, "11th request should be rejected")

	time.Sleep(1100 * time.Millisecond)

	result, err = limiter.AllowWithAlgorithm(ctx, key, limit, window, AlgorithmTokenBucket)
	require.NoError(t, err)
	t.Logf("After refill: allowed=%v, remaining=%d", result.Allowed, result.LimitRemaining)
	assert.True(t, result.Allowed, "Request after refill should be allowed")
}

func TestReset(t *testing.T) {
	limiter, mr := setupTestLimiter(t)
	ctx := context.Background()
	key := "test:reset:key"
	limit := 5
	window := time.Second

	for i := 0; i < limit; i++ {
		_, err := limiter.Allow(ctx, key, limit, window)
		require.NoError(t, err)
	}

	err := limiter.Reset(ctx, key)
	require.NoError(t, err)

	exists := mr.Exists(key)
	assert.False(t, exists, "Key should be deleted after reset")
}

func TestZeroLimit(t *testing.T) {
	limiter, _ := setupTestLimiter(t)
	ctx := context.Background()
	key := "test:zero:key"

	result, err := limiter.Allow(ctx, key, 0, time.Second)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, 0, result.Current)
}
