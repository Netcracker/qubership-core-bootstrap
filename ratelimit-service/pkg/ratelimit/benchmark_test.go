package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupBenchmark(b *testing.B) (*Limiter, func()) {
    mr := miniredis.RunT(b)
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    limiter := NewLimiter(rdb, nil)
    return limiter, func() {
        rdb.Close()
        mr.Close()
    }
}

func BenchmarkFixedWindow(b *testing.B) {
    limiter, cleanup := setupBenchmark(b)
    defer cleanup()
    ctx := context.Background()
    key := "bench:fixed"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        limiter.AllowWithAlgorithm(ctx, key, 10000, time.Second, AlgorithmFixedWindow)
    }
}

func BenchmarkSlidingWindow(b *testing.B) {
    limiter, cleanup := setupBenchmark(b)
    defer cleanup()
    ctx := context.Background()
    key := "bench:sliding"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        limiter.AllowWithAlgorithm(ctx, key, 10000, time.Second, AlgorithmSlidingWindow)
    }
}

func BenchmarkTokenBucket(b *testing.B) {
    limiter, cleanup := setupBenchmark(b)
    defer cleanup()
    ctx := context.Background()
    key := "bench:token"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        limiter.AllowWithAlgorithm(ctx, key, 10000, time.Second, AlgorithmTokenBucket)
    }
}