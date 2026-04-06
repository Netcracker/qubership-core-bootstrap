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
)

var namespace = utils.GetEnv("NAMESPACE", "core-1-core")

func TestCloudE2E_RateLimitThroughGateway(t *testing.T) {
    // 1. Setup
    t.Log("=== Setting up cloud E2E test ===")

    // Port-forwards
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
    userID := "cloud-e2e-user"

    // 2. Clean up any existing rules first
    t.Log("\n=== Cleaning up existing rules ===")
    deleteRule(operatorURL, "e2e_test_rule")
	t.Log("\n=== Cleaning up existing keys for cloud-e2e-user ===")
	cleanupRedisKeys("cloud-e2e-user")

    // 3. Test default rate limit (60 requests per minute from default rule)
    t.Log("\n=== Testing default rate limit ===")

    for i := 0; i < 3; i++ {
        statusCode, err := sendGatewayRequest(gatewayURL, "/test", userID)
        require.NoError(t, err)
        t.Logf("Request %d: HTTP %d", i+1, statusCode)
        assert.Equal(t, 200, statusCode, "Default rule should allow 60 requests per minute")
        time.Sleep(100 * time.Millisecond)
    }

    // 4. Add custom rule with stricter limit
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

    // 5. Test custom rate limit
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

    // 6. Get violating users BEFORE reset (should show the user)
    t.Log("\n=== Getting violating users (before reset) ===")

    violatingUsersJSON, err := getViolatingUsersRaw(operatorURL)
    require.NoError(t, err)
    t.Logf("Violating users API response:\n%s", violatingUsersJSON)

    // 7. Test rate limit reset
    t.Log("\n=== Testing rate limit reset ===")

    err = resetUserRateLimit(operatorURL, userID)
    require.NoError(t, err)
    t.Log("Rate limits reset")

    time.Sleep(1 * time.Second)

	// 8. After reset, request should be allowed immediately
    statusCode, err := sendGatewayRequest(gatewayURL, "/test", userID)
    require.NoError(t, err)
    t.Logf("Request after reset: HTTP %d", statusCode)
    assert.Equal(t, 200, statusCode, "Request after reset should be allowed")
    
    time.Sleep(2 * time.Second)
    
    // 9. Get violating users AFTER reset (should be empty)
    t.Log("\n=== Getting violating users (after reset) ===")

    violatingUsersAfter, err := getViolatingUsers(operatorURL)
    require.NoError(t, err)
    t.Logf("Violating users after reset: %v", violatingUsersAfter)
    assert.Empty(t, violatingUsersAfter, "Should be no violating users after reset")

	t.Log("\n=== Getting violating users via API (after reset) ===")
    violatingUsersJSONAfter, err := getViolatingUsersRaw(operatorURL)
    require.NoError(t, err)
    t.Logf("Violating users API response after reset:\n%s", violatingUsersJSONAfter)

    // 10. Test Redis keys
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

    // 11. Clean up
    t.Log("\n=== Cleaning up ===")
    deleteRule(operatorURL, "e2e_test_rule")

    t.Log("\n=== ✅ Cloud E2E test completed successfully! ===")
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