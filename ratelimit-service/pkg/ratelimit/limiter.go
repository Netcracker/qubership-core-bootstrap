package ratelimit

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/go-redis/redis_rate/v10"
    "github.com/redis/go-redis/v9"
    "k8s.io/klog/v2"
)

type Limiter struct {
    limiter *redis_rate.Limiter
    redis   *redis.Client
    metrics MetricsRecorder
}

type MetricsRecorder interface {
    RecordRateLimit(key string, allowed bool, current int, limit int)
}

type Result struct {
    Allowed        bool          `json:"allowed"`
    Current        int           `json:"current"`
    Limit          int           `json:"limit"`
    Window         time.Duration `json:"window"`
    RetryAfter     time.Duration `json:"retry_after,omitempty"`
    LimitRemaining int           `json:"limit_remaining,omitempty"`
}

type Algorithm string

const (
    AlgorithmFixedWindow   Algorithm = "fixed_window"
    AlgorithmSlidingWindow Algorithm = "sliding_window"
    AlgorithmTokenBucket   Algorithm = "token_bucket"
)

func NewLimiter(rdb *redis.Client, metrics MetricsRecorder) *Limiter {
    return &Limiter{
        limiter: redis_rate.NewLimiter(rdb),
        redis:   rdb,
        metrics: metrics,
    }
}

func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (*Result, error) {
    return l.AllowWithAlgorithm(ctx, key, limit, window, AlgorithmSlidingWindow)
}

func (l *Limiter) AllowWithAlgorithm(ctx context.Context, key string, limit int, window time.Duration, algo Algorithm) (*Result, error) {
    if limit <= 0 {
        return &Result{Allowed: true, Limit: limit, Current: 0, Window: window}, nil
    }

    var rate redis_rate.Limit
    switch algo {
    case AlgorithmFixedWindow:
        rate = redis_rate.Limit{
            Rate:   limit,
            Burst:  limit,
            Period: window,
        }
    case AlgorithmTokenBucket:
        rate = redis_rate.Limit{
            Rate:   limit,
            Burst:  limit,
            Period: time.Second,
        }
    default:
        // sliding window
        rate = redis_rate.Limit{
            Rate:   limit,
            Burst:  limit,
            Period: window,
        }
    }

    res, err := l.limiter.Allow(ctx, key, rate)
    if err != nil {
        return nil, fmt.Errorf("rate limit check failed: %w", err)
    }

    result := &Result{
        Allowed:        res.Allowed > 0,
        Current:        limit - int(res.Remaining),
        Limit:          limit,
        Window:         window,
        LimitRemaining: int(res.Remaining),
        RetryAfter:     res.RetryAfter,
    }

    if !result.Allowed {
        userID := extractUserIDFromKey(key)
        if userID != "" {
            violationKey := fmt.Sprintf("violation:%s", userID)
            if err := l.redis.Set(ctx, violationKey, "1", window).Err(); err != nil {
                klog.Errorf("Failed to record violation for user %s: %v", userID, err)
            } else {
                klog.V(4).Infof("Recorded violation for user: %s", userID)
            }
        }
    }


    if l.metrics != nil {
        l.metrics.RecordRateLimit(key, result.Allowed, result.Current, limit)
    }

    if !result.Allowed {
        klog.V(4).Infof("Rate limit exceeded for key %s", key)
    }

    return result, nil
}

func (l *Limiter) Reset(ctx context.Context, key string) error {
    patterns := []string{
        key,                                // fixed window
        key + ":tokens",                    // token bucket
        key + ":rate",                      // sliding window rate
        key + ":burst",                     // burst limit
        key + ":period",                    // period
        key + ":last_access",               // last access time
    }
    
    for _, k := range patterns {
        if err := l.redis.Del(ctx, k).Err(); err != nil && err != redis.Nil {
            return fmt.Errorf("failed to delete key %s: %w", k, err)
        }
    }
    
    if err := l.redis.Del(ctx, key).Err(); err != nil && err != redis.Nil {
        return err
    }
    
    klog.V(4).Infof("Reset rate limiter for key: %s", key)
    return nil
}

func extractUserIDFromKey(key string) string {
    parts := strings.Split(key, "|")
    for _, part := range parts {
        if strings.HasPrefix(part, "user_id=") {
            return strings.TrimPrefix(part, "user_id=")
        }
    }
    return ""
}