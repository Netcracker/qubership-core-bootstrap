//go:build integration
// +build integration

package api

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "ratelimit-service/pkg/ratelimit"
    "github.com/alicebob/miniredis/v2"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestIntegration_APIWithRateLimit(t *testing.T) {
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

    // Add rule
    rule := &ratelimit.Rule{
        Name:      "test",
        Pattern:   ".*user_id=test.*",
        Limit:     2,
        Window:    time.Minute,
        Algorithm: ratelimit.FixedWindow,
    }
    manager.AddRule(rule)

    // Create API server
    server := NewServer(redisClient, nil, manager)
    
    // Test check endpoint
    reqBody := map[string]interface{}{
        "components": map[string]string{
            "user_id": "test",
            "path":    "/api",
        },
    }
    body, _ := json.Marshal(reqBody)
    req := httptest.NewRequest("POST", "/api/v1/ratelimit/check", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    
    server.checkRateLimit(w, req)
    
    assert.Equal(t, http.StatusOK, w.Code)
    
    var response map[string]interface{}
    json.NewDecoder(w.Body).Decode(&response)
    
    allowed, ok := response["allowed"].(bool)
    assert.True(t, ok)
    assert.True(t, allowed)
}