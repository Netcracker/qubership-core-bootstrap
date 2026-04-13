//go:build integration
// +build integration

package controller

import (
    "context"
    "testing"
    "time"

    "ratelimit-service/pkg/ratelimit"
    "github.com/alicebob/miniredis/v2"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "k8s.io/client-go/kubernetes/fake"
)

func TestIntegration_ControllerWithRateLimit(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Create mock Redis
    mr := miniredis.RunT(t)
    defer mr.Close()

    // Create Redis client
    redisClient, err := ratelimit.NewRedisClient(mr.Addr(), "", 0)
    require.NoError(t, err)
    defer redisClient.Close()

    // Create rate limit manager
    manager := ratelimit.NewRateLimitManager(redisClient)
    redisClient.SetManager(manager)

    // Create fake k8s client
    clientset := fake.NewSimpleClientset()

    // Create controller
    controller := NewConfigMapController(clientset, redisClient, manager)

    assert.NotNil(t, controller)
    
    // Test rate limit check
    ctx := context.Background()
    
    // Add rule
    rule := &ratelimit.Rule{
        Name:      "test",
        Pattern:   ".*user_id=test.*",
        Limit:     2,
        Window:    time.Minute,
        Algorithm: ratelimit.FixedWindow,
    }
    manager.AddRule(rule)
    
    // Check rate limit
    allowed, _, err := manager.Check(ctx, "user_id=test")
    require.NoError(t, err)
    assert.True(t, allowed)
}