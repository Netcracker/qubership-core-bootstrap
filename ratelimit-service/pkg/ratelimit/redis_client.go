package ratelimit

import (
    "context"
    "fmt"
    "sort"
    "strconv"
    "strings"
    "time"

    "github.com/redis/go-redis/v9"
    "k8s.io/klog/v2"
)

type RedisClient struct {
    client  *redis.Client
    config  *Config
    limiter *Limiter
    manager *RateLimitManager
}

func NewRedisClient(addr, password string, db int) (*RedisClient, error) {
    rdb := redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: password,
        DB:       db,
    })

    ctx := context.Background()
    if err := rdb.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("failed to connect to Redis: %w", err)
    }

    config := GetDefaultConfig()

    metrics := NewMetrics(nil)
    limiter := NewLimiter(rdb, metrics)
    manager := NewRateLimitManager(limiter)

    return &RedisClient{
        client:  rdb,
        config:  config,
        limiter: limiter,
        manager: manager,
    }, nil
}

func (r *RedisClient) GetConfig() *Config {
    return r.config
}

func (r *RedisClient) SetConfig(config *Config) {
    r.config = config
}

func (r *RedisClient) GetSeparator() string {
    return r.config.Separator
}

func (r *RedisClient) BuildKey(components map[string]string) string {

    keys := make([]string, 0, len(components))
    for k := range components {
        keys = append(keys, k)
    }
    sort.Strings(keys)

    parts := make([]string, 0, len(components))
    for _, k := range keys {
        parts = append(parts, k+"="+components[k])
    }
    return strings.Join(parts, r.config.Separator)
}

func (r *RedisClient) ParseKey(key string) (*LimitKey, error) {
    limitKey := &LimitKey{
        FullKey:    key,
        Components: make(map[string]string),
    }

    // Remove "rate:" prefix if present
    if strings.HasPrefix(key, "rate:") {
        key = strings.TrimPrefix(key, "rate:")
    }

    // Remove domain prefix if present
    if strings.HasPrefix(key, r.config.Domain+"_") {
        limitKey.Domain = r.config.Domain
        key = strings.TrimPrefix(key, r.config.Domain+"_")
    }

    // Split by separator
    parts := strings.Split(key, r.config.Separator)

    for _, part := range parts {
        kv := strings.SplitN(part, "=", 2)
        if len(kv) == 2 {
            limitKey.Components[kv[0]] = kv[1]
        }
    }

    // Find limit from config
    limit, unit, err := r.findLimit(limitKey.Components)
    if err != nil {
        limit, unit, err = r.findGlobalLimit()
        if err != nil {
            return nil, fmt.Errorf("failed to find limit for key %s: %w", key, err)
        }
    }
    limitKey.LimitValue = limit
    limitKey.Unit = unit
    limitKey.TTL = r.getTTLForUnit(unit)

    return limitKey, nil
}

func (r *RedisClient) findLimit(components map[string]string) (int, string, error) {
    return r.matchLimit(r.config.Descriptors, components)
}

func (r *RedisClient) matchLimit(descriptors []RateLimitDescriptor, components map[string]string) (int, string, error) {
    for _, desc := range descriptors {
        if value, exists := components[desc.Key]; exists {
            if desc.Value == "" || desc.Value == value {
                if len(desc.Descriptors) > 0 {
                    limit, unit, err := r.matchLimit(desc.Descriptors, components)
                    if err == nil {
                        return limit, unit, nil
                    }
                }
                if desc.RateLimit != nil {
                    return desc.RateLimit.RequestsPerUnit, desc.RateLimit.Unit, nil
                }
            }
        }
    }
    return 0, "", fmt.Errorf("no matching rate limit found")
}

func (r *RedisClient) findGlobalLimit() (int, string, error) {
    for _, desc := range r.config.Descriptors {
        if desc.Key == "" && desc.RateLimit != nil {
            return desc.RateLimit.RequestsPerUnit, desc.RateLimit.Unit, nil
        }
    }
    return 60, "minute", nil
}

func (r *RedisClient) getTTLForUnit(unit string) time.Duration {
    switch unit {
    case "second":
        return time.Second
    case "minute":
        return time.Minute
    case "hour":
        return time.Hour
    case "day":
        return 24 * time.Hour
    default:
        return time.Minute
    }
}

func (r *RedisClient) GetUserRateLimitInfo(ctx context.Context, userID string) (*UserRateLimitInfo, error) {
    pattern := fmt.Sprintf("*user_id=%s*", userID)
    keys, err := r.client.Keys(ctx, pattern).Result()
    if err != nil {
        return nil, err
    }

    info := &UserRateLimitInfo{
        UserID: userID,
        Limits: make([]LimitInfo, 0),
    }

    for _, key := range keys {
        // Try to get value
        var current int
        valStr, err := r.client.Get(ctx, key).Result()
        if err == nil {
            if i, err := strconv.Atoi(valStr); err == nil {
                current = i
            } else if f, err := strconv.ParseFloat(valStr, 64); err == nil {
                current = int(f)
            }
        } else {
            // Try HGET for token bucket
            valStr, err = r.client.HGet(ctx, key, "tokens").Result()
            if err == nil {
                if f, err := strconv.ParseFloat(valStr, 64); err == nil {
                    current = int(f)
                }
            }
        }

        ttl, _ := r.client.TTL(ctx, key).Result()
        parsedKey, _ := r.ParseKey(key)

        info.Limits = append(info.Limits, LimitInfo{
            Key:        key,
            Current:    current,
            TTL:        ttl,
            LimitValue: parsedKey.LimitValue,
            Unit:       parsedKey.Unit,
            Components: parsedKey.Components,
        })
    }

    return info, nil
}

func (r *RedisClient) GetViolatingUsers(ctx context.Context) ([]ViolatingUser, error) {
    keys, err := r.client.Keys(ctx, "*user_id*").Result()
    if err != nil {
        return nil, err
    }

    klog.V(4).Infof("Found %d keys with user_id", len(keys))
    
    violatingMap := make(map[string]*ViolatingUser)

    for _, key := range keys {
        parsedKey, err := r.ParseKey(key)
        if err != nil {
            continue
        }
        
        userID, hasUser := parsedKey.Components["user_id"]
        if !hasUser {
            continue
        }
        
        count, err := r.getRequestCount(ctx, key)
        if err != nil {
            klog.V(4).Infof("Failed to get count for key %s: %v", key, err)
            continue
        }
        
        limit := parsedKey.LimitValue
        if limit > 0 && count >= limit {
            if _, exists := violatingMap[userID]; !exists {
                violatingMap[userID] = &ViolatingUser{
                    UserID:      userID,
                    Violations:  make([]ViolationDetail, 0),
                    TotalExceed: count - limit,
                }
            }
        }
    }
    
    violating := make([]ViolatingUser, 0, len(violatingMap))
    for _, v := range violatingMap {
        violating = append(violating, *v)
    }
    
    klog.V(4).Infof("Found %d violating users", len(violating))
    return violating, nil
}

func (r *RedisClient) getRequestCount(ctx context.Context, key string) (int, error) {
    count, err := r.client.ZCard(ctx, key).Result()
    if err == nil {
        return int(count), nil
    }
    
    val, err := r.client.Get(ctx, key).Int()
    if err == nil {
        return val, nil
    }
    
    val, err = r.client.HGet(ctx, key, "tokens").Int()
    if err == nil {
        return 0, nil
    }
    
    return 0, fmt.Errorf("unknown key type")
}

func (r *RedisClient) GetAllStatistics(ctx context.Context) (*Statistics, error) {
    keys, err := r.client.Keys(ctx, "*").Result()
    if err != nil {
        return nil, err
    }

    stats := &Statistics{
        TotalKeys: len(keys),
        Limits:    make([]LimitStat, 0),
        ByType:    make(map[string]int),
        ByUser:    make(map[string]int),
    }

    for _, key := range keys {
        val, err := r.client.Get(ctx, key).Int()
        if err != nil {
            continue
        }

        ttl, _ := r.client.TTL(ctx, key).Result()
        parsedKey, err := r.ParseKey(key)
        if err != nil {
            continue
        }

        stat := LimitStat{
            Key:        key,
            Count:      val,
            TTL:        ttl,
            Type:       r.detectType(parsedKey),
            LimitValue: parsedKey.LimitValue,
            Unit:       parsedKey.Unit,
            Components: parsedKey.Components,
        }

        stats.Limits = append(stats.Limits, stat)
        stats.ByType[stat.Type]++

        if userID, ok := parsedKey.Components["user_id"]; ok {
            stats.ByUser[userID]++
        }
    }

    return stats, nil
}

func (r *RedisClient) ResetUserRateLimit(ctx context.Context, userID string) error {
    pattern := fmt.Sprintf("*user_id=%s*", userID)
    keys, err := r.client.Keys(ctx, pattern).Result()
    if err != nil {
        return err
    }
    
    for _, key := range keys {
        if err := r.client.Del(ctx, key).Err(); err != nil {
            klog.Errorf("Failed to delete key %s: %v", key, err)
            continue
        }
        
        cleanKey := strings.TrimPrefix(key, "rate:")
        if err := r.limiter.Reset(ctx, cleanKey); err != nil {
            klog.Errorf("Failed to reset limiter for key %s: %v", cleanKey, err)
        }
        
        klog.V(4).Infof("Reset key: %s", key)
    }
    
    klog.Infof("Reset rate limits for user: %s, deleted %d keys", userID, len(keys))
    return nil
}

func (r *RedisClient) Allow(ctx context.Context, components map[string]string) (*Result, error) {
    if r.manager == nil {
        return nil, fmt.Errorf("rate limit manager not initialized")
    }
    key := r.BuildKey(components)
    return r.manager.Check(ctx, key)
}

func (r *RedisClient) Reset(ctx context.Context, components map[string]string) error {
    if r.limiter == nil {
        return fmt.Errorf("limiter not initialized")
    }
    key := r.BuildKey(components)
    return r.limiter.Reset(ctx, key)
}

func (r *RedisClient) detectType(parsedKey *LimitKey) string {
    if _, ok := parsedKey.Components["path"]; ok {
        if _, ok := parsedKey.Components["user_id"]; ok {
            return "user_path"
        }
        return "path"
    }
    if _, ok := parsedKey.Components["user_id"]; ok {
        return "user"
    }
    if _, ok := parsedKey.Components["source_ip"]; ok {
        return "ip"
    }
    return "unknown"
}

func (r *RedisClient) Close() error {
    return r.client.Close()
}

// Types for API responses
type UserRateLimitInfo struct {
    UserID string      `json:"user_id"`
    Limits []LimitInfo `json:"limits"`
}

type LimitInfo struct {
    Key        string            `json:"key"`
    Current    int               `json:"current"`
    TTL        time.Duration     `json:"ttl"`
    LimitValue int               `json:"limit_value"`
    Unit       string            `json:"unit"`
    Components map[string]string `json:"components,omitempty"`
}

type Statistics struct {
    TotalKeys int            `json:"total_keys"`
    Limits    []LimitStat    `json:"limits"`
    ByType    map[string]int `json:"by_type"`
    ByUser    map[string]int `json:"by_user,omitempty"`
}

type LimitStat struct {
    Key        string            `json:"key"`
    Count      int               `json:"count"`
    TTL        time.Duration     `json:"ttl"`
    Type       string            `json:"type"`
    LimitValue int               `json:"limit_value"`
    Unit       string            `json:"unit"`
    Components map[string]string `json:"components,omitempty"`
}

type ViolatingUser struct {
    UserID      string            `json:"user_id"`
    Violations  []ViolationDetail `json:"violations"`
    TotalExceed int               `json:"total_exceed"`
}

type ViolationDetail struct {
    Key        string            `json:"key"`
    Current    int               `json:"current"`
    Limit      int               `json:"limit"`
    ExceededBy int               `json:"exceeded_by"`
    Unit       string            `json:"unit"`
    Components map[string]string `json:"components,omitempty"`
}
