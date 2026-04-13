//go:build integration
// +build integration

package ratelimit

import (
    "context"
    "testing"
    "time"

    "github.com/alicebob/miniredis/v2"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestIntegration_RateLimit(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Create mock Redis
    mr := miniredis.RunT(t)
    defer mr.Close()

    // Create Redis client
    redisClient, err := NewRedisClient(mr.Addr(), "", 0)
    require.NoError(t, err)
    defer redisClient.Close()

    // Create rate limit manager
    manager := NewRateLimitManager(redisClient)
    redisClient.SetManager(manager)

    // Add rule
    rule := &Rule{
        Name:      "test",
        Pattern:   ".*user_id=test.*",
        Limit:     2,
        Window:    time.Minute,
        Algorithm: FixedWindow,
    }
    err = manager.AddRule(rule)
    require.NoError(t, err)

    ctx := context.Background()
    components := map[string]string{
        "user_id": "test",
        "path":    "/api",
    }

    // First request should be allowed
    result, err := manager.CheckWithComponents(ctx, components, "|")
    require.NoError(t, err)
    assert.True(t, result["allowed"].(bool))
    assert.Equal(t, 2, result["limit"])

    // Second request should be allowed
    result, err = manager.CheckWithComponents(ctx, components, "|")
    require.NoError(t, err)
    assert.True(t, result["allowed"].(bool))

    // Third request should be rejected
    result, err = manager.CheckWithComponents(ctx, components, "|")
    require.NoError(t, err)
    assert.False(t, result["allowed"].(bool))
}

func TestIntegration_GetViolatingUsers(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    mr := miniredis.RunT(t)
    defer mr.Close()

    redisClient, err := NewRedisClient(mr.Addr(), "", 0)
    require.NoError(t, err)
    defer redisClient.Close()

    manager := NewRateLimitManager(redisClient)
    redisClient.SetManager(manager)

    rule := &Rule{
        Name:      "test",
        Pattern:   ".*user_id=.*",
        Limit:     1,
        Window:    time.Minute,
        Algorithm: FixedWindow,
    }
    manager.AddRule(rule)

    ctx := context.Background()

    // Create rate limited user
    components1 := map[string]string{"user_id": "user1", "path": "/api"}
    manager.CheckWithComponents(ctx, components1, "|") // 1st request
    manager.CheckWithComponents(ctx, components1, "|") // 2nd request - rate limited

    // Create another user within limit
    components2 := map[string]string{"user_id": "user2", "path": "/api"}
    manager.CheckWithComponents(ctx, components2, "|") // 1st request only

    // Get violating users
    users, err := redisClient.GetViolatingUsers(ctx)
    require.NoError(t, err)
    
    // Should have at least one violating user
    assert.GreaterOrEqual(t, len(users), 1)
}