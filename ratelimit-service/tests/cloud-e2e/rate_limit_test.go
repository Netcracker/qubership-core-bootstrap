//go:build cloud_e2e
// +build cloud_e2e

package cloud_e2e

import (
    "bytes"
	"strings"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os/exec"
    "ratelimit-service/pkg/utils"
    "testing"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

const (
    gatewayPort  = "8080"
    operatorPort = "8082"
	metricsPort  = "9090"
)

var namespace = utils.GetEnv("NAMESPACE", "core-1-core")

func TestCloudE2E_RateLimitThroughGateway(t *testing.T) {
    t.Log("=== Setting up cloud E2E test ===")

    t.Log("Starting port-forwards...")
    gatewayPF := setupPortForward(t, "svc/public-gateway-istio", gatewayPort)
    defer gatewayPF()
    operatorPF := setupPortForward(t, "svc/ratelimit-service", operatorPort)
    defer operatorPF()
    redisPF := setupPortForward(t, "svc/redis", "6379")
    defer redisPF()
	metricsPF := setupPortForward(t, "svc/ratelimit-service", metricsPort) 
	defer metricsPF()

    time.Sleep(5 * time.Second)

    gatewayURL := fmt.Sprintf("http://localhost:%s", gatewayPort)
    operatorURL := fmt.Sprintf("http://localhost:%s", operatorPort)
	metricsURL := fmt.Sprintf("http://localhost:%s", metricsPort)
    userID := "cloud-e2e-user"

    t.Log("\n=== Cleaning up existing rules ===")
    deleteRule(operatorURL, "e2e_test_rule")
	t.Log("\n=== Cleaning up existing keys for cloud-e2e-user ===")
	cleanupRedisKeys("cloud-e2e-user")

    t.Log("\n=== Testing default rate limit ===")

    for i := 0; i < 3; i++ {
        statusCode, err := sendGatewayRequest(gatewayURL, "/test", userID)
        require.NoError(t, err)
        t.Logf("Request %d: HTTP %d", i+1, statusCode)
        assert.Equal(t, 200, statusCode, "Default rule should allow 60 requests per minute")
        time.Sleep(100 * time.Millisecond)
    }

    t.Log("\n=== Adding custom rule: 2 requests per 60 seconds ===")

    rule := map[string]interface{}{
        "name":       "e2e_test_rule",
        "pattern":    "/test",
        "limit":      2,
        "window_sec": 60,
        "algorithm":  "fixed_window",
    }

    body, _ := json.Marshal(rule)
    resp, err := http.Post(operatorURL+"/api/v1/ratelimit/rules", "application/json", bytes.NewBuffer(body))
    require.NoError(t, err)
    resp.Body.Close()
    t.Log("Rate limit rule added")

    // Wait for rule to be applied
    time.Sleep(3 * time.Second)

    t.Log("\n=== Testing custom rate limit (2 requests per 10 seconds) ===")

    for i := 0; i < 3; i++ {
        statusCode, err := sendGatewayRequest(gatewayURL, "/test", userID)
        require.NoError(t, err)
        t.Logf("Request %d: HTTP %d", i+1, statusCode)
        if i < 2 {
            assert.Equal(t, 200, statusCode, "First 2 requests should be allowed")
        } else {
            assert.Equal(t, 429, statusCode, "Third request should be rate limited")
        }
        time.Sleep(100 * time.Millisecond)
    }

    t.Log("\n=== Getting violating users (before reset) ===")

    violatingUsersJSON, err := getViolatingUsersRaw(operatorURL)
    require.NoError(t, err)
    t.Logf("Violating users API response:\n%s", violatingUsersJSON)

    t.Log("\n=== Testing rate limit reset ===")

    err = resetUserRateLimit(operatorURL, userID)
    require.NoError(t, err)
    t.Log("Rate limits reset")

    time.Sleep(1 * time.Second)

	//  After reset, request should be allowed immediately
    statusCode, err := sendGatewayRequest(gatewayURL, "/test", userID)
    require.NoError(t, err)
    t.Logf("Request after reset: HTTP %d", statusCode)
    assert.Equal(t, 200, statusCode, "Request after reset should be allowed")
    
    time.Sleep(2 * time.Second)
    
    t.Log("\n=== Getting violating users (after reset) ===")

    violatingUsersAfter, err := getViolatingUsers(operatorURL)
    require.NoError(t, err)
    t.Logf("Violating users after reset: %v", violatingUsersAfter)
    assert.Empty(t, violatingUsersAfter, "Should be no violating users after reset")

	t.Log("\n=== Getting violating users via API (after reset) ===")
    violatingUsersJSONAfter, err := getViolatingUsersRaw(operatorURL)
    require.NoError(t, err)
    t.Logf("Violating users API response after reset:\n%s", violatingUsersJSONAfter)

    t.Log("\n=== Testing Redis keys ===")

    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
        DB:   0,
    })
    defer rdb.Close()

    keys, err := rdb.Keys(context.Background(), "*").Result()
    require.NoError(t, err)
    t.Logf("Redis keys found: %d", len(keys))
    for _, key := range keys {
        val, _ := rdb.Get(context.Background(), key).Result()
        t.Logf("  Key: %s = %s", key, val)
    }

    t.Log("\n=== Testing metrics ===")
    
    metrics, err := getMetrics(metricsURL)
    require.NoError(t, err)
    t.Logf("Metrics sample:\n%s", metrics[:min(500, len(metrics))])

    t.Log("\n=== Cleaning up ===")
    deleteRule(operatorURL, "e2e_test_rule")

    t.Log("\n=== ✅ Cloud E2E test completed successfully! ===")
}

func TestCloudE2E_TwoUsersRateLimit(t *testing.T) {
    t.Log("=== Setting up Two Users Rate Limit Test ===")

    t.Log("Starting port-forwards...")
    gatewayPF := setupPortForward(t, "svc/public-gateway-istio", gatewayPort)
    defer gatewayPF()
    operatorPF := setupPortForward(t, "svc/ratelimit-service", operatorPort)
    defer operatorPF()
    redisPF := setupPortForward(t, "svc/redis", "6379")
    defer redisPF()

    time.Sleep(5 * time.Second)

    gatewayURL := fmt.Sprintf("http://localhost:%s", gatewayPort)
    operatorURL := fmt.Sprintf("http://localhost:%s", operatorPort)
    goodUser := "good-user"
    badUser := "bad-user"

    t.Log("\n=== Cleaning up existing rules ===")
    deleteRule(operatorURL, "bad_user_rule")

    t.Log("\n=== Adding rate limit rule for BAD user: 30 requests per minute ===")

    rule := map[string]interface{}{
        "name":       "bad_user_rule",
        "pattern":    ".*user_id=bad-user.*",
        "limit":      30,
        "window_sec": 60,
        "algorithm":  "fixed_window",
    }

    body, _ := json.Marshal(rule)
    resp, err := http.Post(operatorURL+"/api/v1/ratelimit/rules", "application/json", bytes.NewBuffer(body))
    require.NoError(t, err)
    resp.Body.Close()
    t.Log("Rate limit rule added for bad user only (pattern: .*user_id=bad-user.*)")

    time.Sleep(3 * time.Second)

    t.Log("\n=== Verifying rule applied ===")
    
    checkResp, err := http.Post(operatorURL+"/api/v1/ratelimit/check", "application/json", 
        bytes.NewBuffer([]byte(`{"components":{"path":"/test","user_id":"bad-user"}}`)))
    if err == nil {
        var result map[string]interface{}
        json.NewDecoder(checkResp.Body).Decode(&result)
        t.Logf("Bad user limit: %v", result["limit"])
        checkResp.Body.Close()
    }
    
    checkResp, err = http.Post(operatorURL+"/api/v1/ratelimit/check", "application/json", 
        bytes.NewBuffer([]byte(`{"components":{"path":"/test","user_id":"good-user"}}`)))
    if err == nil {
        var result map[string]interface{}
        json.NewDecoder(checkResp.Body).Decode(&result)
        t.Logf("Good user limit: %v", result["limit"])
        checkResp.Body.Close()
    }

    t.Log("\n=== Testing GOOD user (no rate limit applied) ===")
    
    goodSuccess := 0
    goodLimited := 0
    requests := 50

    for i := 0; i < requests; i++ {
        statusCode, err := sendGatewayRequest(gatewayURL, "/test", goodUser)
        require.NoError(t, err)
        if statusCode == 200 {
            goodSuccess++
        } else if statusCode == 429 {
            goodLimited++
        }
        if (i+1)%10 == 0 {
            t.Logf("Good user progress: %d/%d requests", i+1, requests)
        }
        time.Sleep(50 * time.Millisecond)
    }

    t.Logf("\nGood user results: %d OK, %d Rate Limited out of %d requests", goodSuccess, goodLimited, requests)
    
    assert.Equal(t, 0, goodLimited, "Good user should NOT be rate limited (got %d)", goodLimited)

    t.Log("\n=== Testing BAD user (rate limited after ~30 requests) ===")

    badSuccess := 0
    badLimited := 0

    for i := 0; i < requests; i++ {
        statusCode, err := sendGatewayRequest(gatewayURL, "/test", badUser)
        require.NoError(t, err)
        if statusCode == 200 {
            badSuccess++
        } else if statusCode == 429 {
            badLimited++
        }
        
        if badLimited == 1 && i > 0 {
            t.Logf("Bad user rate limited started at request #%d", i+1)
        }
        
        if (i+1)%10 == 0 {
            t.Logf("Bad user progress: %d/%d requests", i+1, requests)
        }
        time.Sleep(50 * time.Millisecond)
    }

    t.Logf("\nBad user results: %d OK, %d Rate Limited out of %d requests", badSuccess, badLimited, requests)

    // Expected: ~30 OK, ~20 Limited
    expectedLimited := requests - 30
    if expectedLimited < 0 {
        expectedLimited = 0
    }
    
    t.Logf("Expected bad user limited requests: ~%d (50 total - 30 limit)", expectedLimited)
    t.Logf("Actual bad user limited requests: %d", badLimited)
    
    // Allow some tolerance (within 10 requests due to network delays)
    assert.InDelta(t, expectedLimited, badLimited, 10, 
        "Bad user should be rate limited around %d requests (got %d)", expectedLimited, badLimited)
    
    t.Log("\n=== Isolation Check ===")
    if goodLimited == 0 && badLimited > 0 {
        t.Log("✅ PASS: Rate limits are properly isolated per user")
        t.Logf("   Good user: %d limited, Bad user: %d limited", goodLimited, badLimited)
    } else if goodLimited > 0 {
        t.Log("⚠️ WARNING: Good user was also rate limited - isolation may be compromised")
        t.Logf("   Good user: %d limited, Bad user: %d limited", goodLimited, badLimited)
    }

    t.Log("\n=== Getting violating users ===")
    
    violatingUsers, err := getViolatingUsers(operatorURL)
    require.NoError(t, err)
    t.Logf("Violating users: %v", violatingUsers)
    
    // Bad user should be in violating list if rate limited
    if badLimited > 0 {
        assert.Contains(t, violatingUsers, badUser, "Bad user should be in violating users list")
    }
    
    t.Log("\n=== Resetting rate limits for bad user ===")
    
    err = resetUserRateLimit(operatorURL, badUser)
    require.NoError(t, err)
    t.Log("Rate limits reset for bad user")
    
    time.Sleep(2 * time.Second)
    
    // Verify reset - bad user should be allowed again
    t.Log("\n=== Verifying reset ===")
    
    resetSuccess := 0
    for i := 0; i < 10; i++ {
        statusCode, err := sendGatewayRequest(gatewayURL, "/test", badUser)
        require.NoError(t, err)
        if statusCode == 200 {
            resetSuccess++
        }
        time.Sleep(100 * time.Millisecond)
    }
    
    t.Logf("Bad user after reset: %d/10 requests successful", resetSuccess)
    assert.Greater(t, resetSuccess, 8, "After reset, bad user should be able to make requests again")
    
    t.Log("\n=== Cleaning up ===")
    deleteRule(operatorURL, "bad_user_rule")

    t.Log("\n=== ✅ Two Users Rate Limit Test completed successfully! ===")
}


// Helper functions

func setupPortForward(t *testing.T, resource, port string) func() {
    cmd := exec.Command("kubectl", "port-forward", "-n", namespace, resource, port+":"+port)
    if err := cmd.Start(); err != nil {
        t.Logf("Failed to start port-forward for %s: %v", resource, err)
        return func() {}
    }
    time.Sleep(2 * time.Second)
    return func() {
        cmd.Process.Kill()
    }
}

func sendGatewayRequest(gatewayURL, endpoint, userID string) (int, error) {
    url := gatewayURL + endpoint
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return 0, err
    }
    req.Header.Set("x-user-id", userID)

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()
    return resp.StatusCode, nil
}

func getViolatingUsersRaw(apiURL string) (string, error) {
    resp, err := http.Get(apiURL + "/api/v1/users/violating")
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    // Pretty print JSON
    var prettyJSON bytes.Buffer
    if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
        return string(body), nil
    }

    return prettyJSON.String(), nil
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

func deleteRule(apiURL, ruleName string) {
    req, err := http.NewRequest("DELETE", apiURL+"/api/v1/ratelimit/rules/"+ruleName, nil)
    if err != nil {
        return
    }
    client := &http.Client{}
    resp, err := client.Do(req)
    if err == nil {
        resp.Body.Close()
    }
}


func cleanupRedisKeys(userID string) {
    cmd := exec.Command("kubectl", "exec", "-n", namespace, "deployment/redis", "--",
        "redis-cli", "KEYS", "*user_id*")
    output, _ := cmd.Output()
    keys := strings.Split(string(output), "\n")
    
    for _, key := range keys {
        if key != "" {
            delCmd := exec.Command("kubectl", "exec", "-n",  namespace, "deployment/redis", "--",
                "redis-cli", "DEL", key)
            delCmd.Run()
        }
    }
}

func getMetrics(apiURL string) (string, error) {
    metricsURL := fmt.Sprintf("http://localhost:%s/metrics", metricsPort)
    resp, err := http.Get(metricsURL)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }
    
    lines := strings.Split(string(body), "\n")
    var ourMetrics []string
    
    for _, line := range lines {
        if strings.Contains(line, "ratelimit_") {
            ourMetrics = append(ourMetrics, line)
        }
    }
    
    if len(ourMetrics) == 0 {
        return "No rate limit metrics found", nil
    }
    
    return "=== RateLimit Metrics ===\n" + 
           strings.Join(ourMetrics, "\n") + 
           "\n=== End Metrics ===", nil
}