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

func setupTestRedisClient(t *testing.T) (*RedisClient, *miniredis.Miniredis) {
    mr := miniredis.RunT(t)
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    
    config := GetDefaultConfig()
    
    metrics := NewMetrics(nil)
    limiter := NewLimiter(rdb, metrics)
    manager := NewRateLimitManager(limiter)
    
    client := &RedisClient{
        client:  rdb,
        config:  config,
        limiter: limiter,
        manager: manager,
    }
    
    t.Cleanup(func() {
        rdb.Close()
        mr.Close()
    })
    
    return client, mr
}

func TestBuildKey(t *testing.T) {
    client, _ := setupTestRedisClient(t)
    
    components := map[string]string{
        "path":    "/test",
        "user_id": "alice",
    }
    
    key := client.BuildKey(components)
    expected := "path=/test|user_id=alice"
    assert.Equal(t, expected, key)
}

func TestParseKey(t *testing.T) {
    client, _ := setupTestRedisClient(t)
    
    key := "path=/test|user_id=alice"
    parsed, err := client.ParseKey(key)
    require.NoError(t, err)
    
    assert.Equal(t, "/test", parsed.Components["path"])
    assert.Equal(t, "alice", parsed.Components["user_id"])
}

func TestParseKeyWithEscaping(t *testing.T) {
    client, _ := setupTestRedisClient(t)
    
    components := map[string]string{
        "user_id": "alice_bob",
    }
    
    key := client.BuildKey(components)
    t.Logf("Built key: %s", key)
    
    parsed, err := client.ParseKey(key)
    require.NoError(t, err)
    
    assert.Equal(t, "alice_bob", parsed.Components["user_id"])
}

func TestGetViolatingUsers(t *testing.T) {
    client, mr := setupTestRedisClient(t)
    ctx := context.Background()
    
    testKeys := map[string]string{
        "path=/test|user_id=alice": "45",
        "user_id=bob":              "75",
        "path=/test|user_id=charlie": "1",
    }
    
    for key, value := range testKeys {
        mr.Set(key, value)
        mr.SetTTL(key, time.Minute)
    }
    
    violating, err := client.GetViolatingUsers(ctx)
    require.NoError(t, err)
    
    assert.GreaterOrEqual(t, len(violating), 1)
    t.Logf("Found %d violating users", len(violating))
}

func TestResetUserRateLimit(t *testing.T) {
    client, mr := setupTestRedisClient(t)
    ctx := context.Background()
    
    testKeys := []string{
        "path=/test|user_id=alice",
        "user_id=alice",
        "user_id=bob",
    }
    
    for _, key := range testKeys {
        mr.Set(key, "10")
        mr.SetTTL(key, time.Minute)
    }
    
    err := client.ResetUserRateLimit(ctx, "alice")
    require.NoError(t, err)
    
    assert.False(t, mr.Exists("path=/test|user_id=alice"))
    assert.False(t, mr.Exists("user_id=alice"))
    assert.True(t, mr.Exists("user_id=bob"))
}

func TestGetAllStatistics(t *testing.T) {
    client, mr := setupTestRedisClient(t)
    ctx := context.Background()
    
    testKeys := map[string]string{
        "path=/test|user_id=alice": "10",
        "user_id=bob":              "5",
        "path=/api":                "8",
    }
    
    for key, value := range testKeys {
        mr.Set(key, value)
        mr.SetTTL(key, time.Minute)
    }
    
    stats, err := client.GetAllStatistics(ctx)
    require.NoError(t, err)
    
    assert.Equal(t, 3, stats.TotalKeys)
    t.Logf("Statistics: Total keys=%d", stats.TotalKeys)
}

func TestGetUserRateLimitInfo(t *testing.T) {
    client, mr := setupTestRedisClient(t)
    ctx := context.Background()
    
    testKeys := map[string]string{
        "path=/test|user_id=alice": "15",
        "path=/api|user_id=alice":  "25",
        "user_id=alice":            "10",
    }
    
    for key, value := range testKeys {
        mr.Set(key, value)
        mr.SetTTL(key, time.Minute)
    }
    
    info, err := client.GetUserRateLimitInfo(ctx, "alice")
    require.NoError(t, err)
    
    assert.Equal(t, "alice", info.UserID)
    assert.Len(t, info.Limits, 3)
}