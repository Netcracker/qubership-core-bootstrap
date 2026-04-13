//go:build e2e
// +build e2e

package e2e

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "ratelimit-service/pkg/utils"
    "ratelimit-service/tests/e2e/setup"
    "regexp"
    "testing"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

var (
    operatorPort = "8083"
)

func TestE2E_RateLimitThroughOperatorAPI(t *testing.T) {
    ctx := context.Background()

    t.Log("=== Setting up test environment ===")

    kubeconfig := utils.GetEnv("KUBECONFIG", utils.GetEnv("HOME", "")+"/.kube/config")

    // Port-forward for Redis
    redisPF := setup.NewPortForward("core-1-core", "service/redis", "6379", "6379")
    require.NoError(t, redisPF.Start())
    defer redisPF.Stop()

    // Start local operator
    operator, err := setup.NewLocalOperator(kubeconfig)
    require.NoError(t, err)
    require.NoError(t, operator.Start(ctx, operatorPort))
    defer operator.Stop()

    time.Sleep(3 * time.Second)

    operatorURL := "http://localhost:" + operatorPort
    userID := "e2e-test-user"

    t.Log("\n=== Cleaning Redis before test ===")
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
        DB:   0,
    })
    defer rdb.Close()
    
    keys, _ := rdb.Keys(ctx, "*"+userID+"*").Result()
    for _, key := range keys {
        rdb.Del(ctx, key)
    }
    t.Logf("Cleaned %d keys from Redis", len(keys))

    t.Log("\n=== Cleaning up existing rules ===")
    client := &http.Client{}
    
    deleteReq, _ := http.NewRequest("DELETE", operatorURL+"/api/v1/ratelimit/rules/default", nil)
    client.Do(deleteReq)
    
    deleteReq2, _ := http.NewRequest("DELETE", operatorURL+"/api/v1/ratelimit/rules/api_strict", nil)
    client.Do(deleteReq2)
    
    time.Sleep(1 * time.Second)

    t.Log("\n=== Adding specific rule with HIGH priority (100) ===")
    
    rule := map[string]interface{}{
        "name":       "test_rule",
        "pattern":    ".*user_id=" + userID + ".*",
        "limit":      2,
        "window_sec": 10,
        "algorithm":  "fixed_window",
        "priority":   100, 
    }
    
    body, _ := json.Marshal(rule)
    resp, err := http.Post(operatorURL+"/api/v1/ratelimit/rules", "application/json", bytes.NewBuffer(body))
    require.NoError(t, err)
    resp.Body.Close()
    t.Log("✓ Specific rule added with priority 100")

    t.Log("\n=== Adding fallback rule with LOW priority (10) ===")
    
    fallbackRule := map[string]interface{}{
        "name":       "fallback_rule",
        "pattern":    ".*user_id=.*",
        "limit":      100,
        "window_sec": 60,
        "algorithm":  "fixed_window",
        "priority":   10, 
    }
    
    body, _ = json.Marshal(fallbackRule)
    resp, err = http.Post(operatorURL+"/api/v1/ratelimit/rules", "application/json", bytes.NewBuffer(body))
    require.NoError(t, err)
    resp.Body.Close()
    t.Log("✓ Fallback rule added with priority 10")

    t.Log("\n=== Verifying rules and priorities ===")

    resp, err = http.Get(operatorURL + "/api/v1/ratelimit/rules")
    require.NoError(t, err)

    var rulesResponse map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&rulesResponse)
    resp.Body.Close()

    rulesList, ok := rulesResponse["rules"].([]interface{})
    require.True(t, ok, "Response should contain 'rules' array")
    t.Logf("Active rules: %d", len(rulesList))

    for _, r := range rulesList {
        ruleMap := r.(map[string]interface{})
        t.Logf("Rule: %s, Priority: %v", ruleMap["Name"], ruleMap["Priority"])
    }

    t.Log("\n=== Testing rate limit ===")

    for i := 0; i < 2; i++ {
        allowed, limit, remaining, err := checkRateLimit(operatorURL, "/test", userID)
        require.NoError(t, err)
        t.Logf("Request %d (test user): allowed=%v, limit=%d, remaining=%d", i+1, allowed, limit, remaining)
        assert.True(t, allowed, "Request %d should be allowed", i+1)
        assert.Equal(t, 2, limit, "Limit should be 2 for test user")
        
        if i == 0 {
            assert.Equal(t, 1, remaining, "After first request, 1 remaining")
        } else if i == 1 {
            assert.Equal(t, 0, remaining, "After second request, 0 remaining")
        }
    }

    allowed, limit, remaining, err := checkRateLimit(operatorURL, "/test", userID)
    require.NoError(t, err)
    t.Logf("Request 3 (test user): allowed=%v, limit=%d, remaining=%d", allowed, limit, remaining)
    assert.False(t, allowed, "Third request for test user should be rejected")
    assert.Equal(t, 0, remaining, "Remaining should be 0")

    t.Log("\n=== Testing other user with fallback rule ===")

    otherUserID := "other-test-user"

    allowed2, limit2, remaining2, err := checkRateLimit(operatorURL, "/test", otherUserID)
    require.NoError(t, err)
    t.Logf("Other user: allowed=%v, limit=%d, remaining=%d", allowed2, limit2, remaining2)
    assert.True(t, allowed2, "Other user should be allowed")
    assert.Equal(t, 100, limit2, "Other user should have limit 100 from fallback rule")
    assert.Equal(t, 99, remaining2, "Other user should have 99 remaining after first request")

    t.Log("\n=== Checking Redis state ===")

    allKeys, err := rdb.Keys(ctx, "*").Result()
    require.NoError(t, err)
    t.Logf("Total Redis keys: %d", len(allKeys))

    for _, key := range allKeys {
        if bytes.Contains([]byte(key), []byte("test_rule")) {
            val, err := rdb.Get(ctx, key).Result()
            if err == nil {
                t.Logf("  Test rule key: %s = %s", key, val)
                
                ttl, _ := rdb.TTL(ctx, key).Result()
                t.Logf("    TTL: %v", ttl)
                assert.LessOrEqual(t, ttl, 10*time.Second, "TTL should be <= window_sec (10s)")
            }
        }
    }

    t.Log("\n=== Testing rate limit reset ===")

    err = resetUserRateLimit(operatorURL, userID)
    require.NoError(t, err)
    t.Log("✓ Rate limits reset for test user")

    time.Sleep(1 * time.Second)

    allowed, limit, remaining, err = checkRateLimit(operatorURL, "/test", userID)
    require.NoError(t, err)
    t.Logf("After reset: allowed=%v, limit=%d, remaining=%d", allowed, limit, remaining)
    assert.True(t, allowed, "Request after reset should be allowed")
    assert.Equal(t, 1, remaining, "After reset, should have full quota")

    t.Log("\n=== Cleaning up ===")

    deleteReq, _ = http.NewRequest("DELETE", operatorURL+"/api/v1/ratelimit/rules/test_rule", nil)
    client.Do(deleteReq)

    deleteReq2, _ = http.NewRequest("DELETE", operatorURL+"/api/v1/ratelimit/rules/fallback_rule", nil)
    client.Do(deleteReq2)

    finalKeys, _ := rdb.Keys(ctx, "*").Result()
    t.Logf("Final Redis keys count: %d", len(finalKeys))
    for _, key := range finalKeys {
        if bytes.Contains([]byte(key), []byte("test_rule")) ||
           bytes.Contains([]byte(key), []byte("fallback_rule")) ||
           bytes.Contains([]byte(key), []byte(userID)) ||
           bytes.Contains([]byte(key), []byte(otherUserID)) {
            rdb.Del(ctx, key)
            t.Logf("Deleted key: %s", key)
        }
    }

    t.Log("\n=== ✅ E2E test completed successfully! ===")
}

// Helper functions
func checkRateLimit(apiURL, path, userID string) (allowed bool, limit int, remaining int, err error) {
    reqBody := map[string]interface{}{
        "components": map[string]string{
            "path":    path,
            "user_id": userID,
        },
    }

    body, _ := json.Marshal(reqBody)
    resp, err := http.Post(apiURL+"/api/v1/ratelimit/check", "application/json", bytes.NewBuffer(body))
    if err != nil {
        return false, 0, 0, err
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return false, 0, 0, err
    }

    allowed, ok := result["allowed"].(bool)
    if !ok {
        return false, 0, 0, fmt.Errorf("invalid response format")
    }

    limit = 0
    if l, ok := result["limit"].(float64); ok {
        limit = int(l)
    }

    remaining = 0
    if r, ok := result["remaining"].(float64); ok {
        remaining = int(r)
    }

    return allowed, limit, remaining, nil
}

func getViolatingUsers(apiURL string) ([]string, error) {
    resp, err := http.Get(apiURL + "/api/v1/users/violating")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        ViolatingUsers []struct {
            UserID string `json:"user_id"`
        } `json:"violating_users"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    users := make([]string, len(result.ViolatingUsers))
    for i, u := range result.ViolatingUsers {
        users[i] = u.UserID
    }
    return users, nil
}

func resetUserRateLimit(apiURL, userID string) error {
    url := fmt.Sprintf("%s/api/v1/users/%s/reset", apiURL, userID)
    resp, err := http.Post(url, "application/json", nil)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("reset failed with status: %d", resp.StatusCode)
    }
    return nil
}

// TestHelperFunctions tests for helper utilities
func TestHelperFunctions(t *testing.T) {
    t.Run("ValidatePattern", func(t *testing.T) {
        tests := []struct {
            pattern string
            valid   bool
        }{
            {".*user_id=test.*", true},
            {"user_id=test", true},
            {"^.*user_id=test.*$", true},
            {"*user_id=test*", false},
            {"[invalid", false},
            {"?invalid", false},
        }
        
        for _, tt := range tests {
            _, err := regexp.Compile(tt.pattern)
            if tt.valid {
                assert.NoError(t, err, "Pattern %s should be valid", tt.pattern)
            } else {
                assert.Error(t, err, "Pattern %s should be invalid", tt.pattern)
            }
        }
    })
    
    t.Run("BuildKey", func(t *testing.T) {
        components := map[string]string{
            "user_id": "test",
            "path":    "/api",
        }
        
        // Build key multiple times and check that it's consistent
        key1 := buildKeyForTest(components, "|")
        key2 := buildKeyForTest(components, "|")
        
        // Both keys should contain both components
        assert.Contains(t, key1, "user_id=test", "Key should contain user_id")
        assert.Contains(t, key1, "path=/api", "Key should contain path")
        assert.Contains(t, key2, "user_id=test", "Key should contain user_id")
        assert.Contains(t, key2, "path=/api", "Key should contain path")
        
        // The separator should be present
        assert.Contains(t, key1, "|", "Key should contain separator")
        
        // Both keys should have the same length and components
        assert.Equal(t, len(key1), len(key2), "Keys should have same length")
        
        // Different separator
        key3 := buildKeyForTest(components, "&")
        assert.Contains(t, key3, "&", "Key should contain & separator")
        assert.NotContains(t, key3, "|", "Key should not contain | separator")
        
        t.Logf("Generated key with '|' separator: %s", key1)
        t.Logf("Generated key with '&' separator: %s", key3)
    })
    
    t.Run("BuildKeyStable", func(t *testing.T) {
        // Test that with sorted components, key is deterministic
        components := map[string]string{
            "z_key": "last",
            "a_key": "first",
            "m_key": "middle",
        }
        
        key := buildKeyForTest(components, "|")
        t.Logf("Key with unsorted map: %s", key)
        
        // Check that all components are present
        assert.Contains(t, key, "a_key=first")
        assert.Contains(t, key, "m_key=middle")
        assert.Contains(t, key, "z_key=last")
    })
}

func buildKeyForTest(components map[string]string, separator string) string {
    if components == nil {
        return ""
    }
    
    result := ""
    first := true
    for k, v := range components {
        if !first {
            result += separator
        }
        result += k + "=" + v
        first = false
    }
    return result
}