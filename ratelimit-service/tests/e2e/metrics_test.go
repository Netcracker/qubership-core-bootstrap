// tests/e2e/metrics_test.go
//go:build e2e
// +build e2e

package e2e

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "testing"
    "time"

    "ratelimit-service/pkg/utils"
    "ratelimit-service/tests/e2e/setup"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

var (
    metricsPort = "9090"
)

func TestE2E_MetricsEndpoint(t *testing.T) {
    t.Log("=== Setting up metrics test environment ===")

    kubeconfig := utils.GetEnv("KUBECONFIG", utils.GetEnv("HOME", "")+"/.kube/config")

    // Port-forward for Redis
    redisPF := setup.NewPortForward("core-1-core", "service/redis", "6379", "6379")
    require.NoError(t, redisPF.Start())
    defer redisPF.Stop()

    // Start local operator
    operator, err := setup.NewLocalOperator(kubeconfig)
    require.NoError(t, err)
    require.NoError(t, operator.Start(context.Background(), operatorPort))
    defer operator.Stop()

    // Wait for services to start
    time.Sleep(5 * time.Second)

    metricsURL := fmt.Sprintf("http://localhost:%s/metrics", metricsPort)

    t.Log("\n=== Testing metrics endpoint availability ===")

    // Try multiple times to connect
    var resp *http.Response
    for i := 0; i < 5; i++ {
        resp, err = http.Get(metricsURL)
        if err == nil {
            break
        }
        t.Logf("Attempt %d: metrics endpoint not ready yet, waiting...", i+1)
        time.Sleep(2 * time.Second)
    }
    
    require.NoError(t, err, "Metrics endpoint should be available")
    defer resp.Body.Close()

    assert.Equal(t, 200, resp.StatusCode, "Metrics endpoint should return 200")
    
    body, err := io.ReadAll(resp.Body)
    require.NoError(t, err)
    
    metrics := string(body)
    t.Logf("Metrics endpoint accessible, response length: %d bytes", len(metrics))
    
    // Check for expected metrics (some may be zero-initialized)
    expectedMetrics := []string{
        "ratelimit_violating_users_total",
        "ratelimit_active_limits_total",
        "ratelimit_redis_operations_total",
        "ratelimit_config_reloads_total",
        "ratelimit_redis_keys_total",
        "ratelimit_redis_memory_bytes",
        "ratelimit_redis_connected_clients",
        "ratelimit_redis_hit_rate",
    }
    
    missingMetrics := []string{}
    for _, metric := range expectedMetrics {
        if !assert.Contains(t, metrics, metric, "Should contain metric: %s", metric) {
            missingMetrics = append(missingMetrics, metric)
        }
    }
    
    if len(missingMetrics) > 0 {
        t.Logf("Missing metrics: %v", missingMetrics)
        t.Logf("Metrics preview: %s", metrics[:min(500, len(metrics))])
    }
    
    t.Log("✓ Metrics endpoint test completed")
}

func TestE2E_MetricsCollection(t *testing.T) {
    t.Log("=== Testing metrics collection ===")

    kubeconfig := utils.GetEnv("KUBECONFIG", utils.GetEnv("HOME", "")+"/.kube/config")

    // Port-forward for Redis
    redisPF := setup.NewPortForward("core-1-core", "service/redis", "6379", "6379")
    require.NoError(t, redisPF.Start())
    defer redisPF.Stop()

    // Start local operator
    operator, err := setup.NewLocalOperator(kubeconfig)
    require.NoError(t, err)
    require.NoError(t, operator.Start(context.Background(), operatorPort))
    defer operator.Stop()

    time.Sleep(5 * time.Second)

    operatorURL := fmt.Sprintf("http://localhost:%s", operatorPort)
    metricsURL := fmt.Sprintf("http://localhost:%s/metrics", metricsPort)
    userID := "metrics-test-user"

    t.Log("\n=== Adding rate limit rule ===")

    rule := map[string]interface{}{
        "name":       "metrics_test_rule",
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
    t.Log("✓ Rule added")

    t.Log("\n=== Making rate limit requests ===")

    // Make requests to trigger rate limiting
    for i := 0; i < 5; i++ {
        checkReq := map[string]interface{}{
            "components": map[string]string{
                "path":    "/test",
                "user_id": userID,
            },
        }
        body, _ := json.Marshal(checkReq)
        resp, err := http.Post(operatorURL+"/api/v1/ratelimit/check", "application/json", bytes.NewBuffer(body))
        if err != nil {
            t.Logf("Request %d failed: %v", i+1, err)
            continue
        }
        
        var result map[string]interface{}
        if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
            t.Logf("Failed to decode response: %v", err)
            resp.Body.Close()
            continue
        }
        resp.Body.Close()
        
        if allowed, ok := result["allowed"].(bool); ok {
            t.Logf("Request %d: allowed=%v", i+1, allowed)
        } else {
            t.Logf("Request %d: unexpected response format", i+1)
        }
        time.Sleep(100 * time.Millisecond)
    }

    t.Log("\n=== Checking metrics after requests ===")

    // Give time for metrics to update
    time.Sleep(3 * time.Second)

    // Get metrics
    resp, err = http.Get(metricsURL)
    require.NoError(t, err)
    defer resp.Body.Close()

    bodyBytes, err := io.ReadAll(resp.Body)
    require.NoError(t, err)
    metrics := string(bodyBytes)

    t.Logf("Metrics collected (first 800 chars):\n%s", metrics[:min(800, len(metrics))])
    
    // Verify metrics show some activity
    if !assert.Contains(t, metrics, "ratelimit_active_limits_total", "Should have active limits") {
        t.Log("Note: Some metrics may be zero if no rate limits were triggered")
    }
    
    t.Log("✓ Metrics collection test completed")

    // Cleanup
    t.Log("\n=== Cleaning up ===")
    deleteRule(operatorURL, "metrics_test_rule")
}

func TestE2E_MetricsServerOnly(t *testing.T) {
    t.Log("=== Testing metrics server only ===")

    kubeconfig := utils.GetEnv("KUBECONFIG", utils.GetEnv("HOME", "")+"/.kube/config")

    // Port-forward for Redis
    redisPF := setup.NewPortForward("core-1-core", "service/redis", "6379", "6379")
    require.NoError(t, redisPF.Start())
    defer redisPF.Stop()

    // Start local operator
    operator, err := setup.NewLocalOperator(kubeconfig)
    require.NoError(t, err)
    require.NoError(t, operator.Start(context.Background(), operatorPort))
    defer operator.Stop()

    time.Sleep(5 * time.Second)

    metricsURL := fmt.Sprintf("http://localhost:%s/metrics", metricsPort)

    // Test metrics endpoint multiple times
    for i := 0; i < 3; i++ {
        resp, err := http.Get(metricsURL)
        require.NoError(t, err)
        
        body, err := io.ReadAll(resp.Body)
        resp.Body.Close()
        require.NoError(t, err)
        
        metrics := string(body)
        t.Logf("Attempt %d: Got %d bytes of metrics", i+1, len(metrics))
        
        // Check that we have some Redis metrics
        assert.Contains(t, metrics, "ratelimit_redis_connected_clients", "Should have Redis clients metric")
        assert.Contains(t, metrics, "ratelimit_redis_hit_rate", "Should have Redis hit rate metric")
        
        time.Sleep(2 * time.Second)
    }
    
    t.Log("✓ Metrics server test completed")
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}