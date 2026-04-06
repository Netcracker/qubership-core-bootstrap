package cloud_e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"testing"
	"time"

	"ratelimit-service/pkg/utils"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	gatewayPort  = "8080"
	operatorPort = "8082"
)

var namespace    =  utils.GetEnv("NAMESPACE", "core-1-core")

func TestCloudE2E_RateLimitThroughGateway(t *testing.T) {
	ctx := context.Background()

	// 1. Setup
	t.Log("=== Setting up cloud E2E test ===")

	kubeconfig := utils.GetEnv("KUBECONFIG", utils.GetEnv("HOME", "") + "/.kube/config")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	require.NoError(t, err)

	_, kubeErr := kubernetes.NewForConfig(config)
	require.NoError(t, kubeErr)

	// Port-forwards
	t.Log("Starting port-forwards...")

	// Gateway port-forward
	gatewayPF := setupPortForward(t, "service/public-gateway-istio", gatewayPort)
	defer gatewayPF()

	// Operator port-forward
	operatorPF := setupPortForward(t, "service/ratelimit-service", operatorPort)
	defer operatorPF()

	// Redis port-forward
	redisPF := setupPortForward(t, "service/redis", "6379")
	defer redisPF()

	time.Sleep(5 * time.Second)

	operatorURL := fmt.Sprintf("http://localhost:%s", operatorPort)
	gatewayURL := fmt.Sprintf("http://localhost:%s", gatewayPort)

	// 2. Test initial configuration
	t.Log("\n=== Testing initial configuration ===")

	// Check default rate limit
	userID := "cloud-e2e-user"

	// Generate JWT token
	jwtToken, err := generateJWTToken(userID)
	require.NoError(t, err)

	// Test rate limit
	endpoint := "/test"

	t.Log("Testing rate limit: 2 requests per 10 seconds (default rule)")

	// First 2 requests should be allowed
	for i := 0; i < 2; i++ {
		statusCode, err := sendGatewayRequest(gatewayURL, endpoint, jwtToken)
		require.NoError(t, err)
		t.Logf("Request %d: HTTP %d", i+1, statusCode)
		assert.Equal(t, 200, statusCode)
		time.Sleep(100 * time.Millisecond)
	}

	// Third request should be rate limited
	statusCode, err := sendGatewayRequest(gatewayURL, endpoint, jwtToken)
	require.NoError(t, err)
	t.Logf("Request 3: HTTP %d", statusCode)
	assert.Equal(t, 429, statusCode, "Rate limit should trigger")

	// 3. Test configuration change via operator API
	t.Log("\n=== Testing configuration change ===")

	// Update rate limit rule
	rule := map[string]interface{}{
		"name":       "new_rule",
		"pattern":    "/test",
		"limit":      5,
		"window_sec": 60,
		"algorithm":  "sliding_window",
	}

	body, _ := json.Marshal(rule)
	resp, err := http.Post(operatorURL+"/api/v1/ratelimit/rules", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	resp.Body.Close()
	t.Log("Rate limit rule updated: 5 requests per minute")

	time.Sleep(2 * time.Second)

	// Wait for rate limit window to reset
	time.Sleep(11 * time.Second)

	// Test new limit
	t.Log("Testing new rate limit: 5 requests per minute")

	for i := 0; i < 5; i++ {
		statusCode, err := sendGatewayRequest(gatewayURL, endpoint, jwtToken)
		require.NoError(t, err)
		t.Logf("Request %d: HTTP %d", i+1, statusCode)
		assert.Equal(t, 200, statusCode)
		time.Sleep(100 * time.Millisecond)
	}

	// 6th request should be rate limited
	statusCode, err = sendGatewayRequest(gatewayURL, endpoint, jwtToken)
	require.NoError(t, err)
	t.Logf("Request 6: HTTP %d", statusCode)
	assert.Equal(t, 429, statusCode, "Should be rate limited after 5 requests")

	// 4. Test separator in keys
	t.Log("\n=== Testing key separator ===")

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	defer rdb.Close()

	// Check Redis key format
	key := fmt.Sprintf("path=%s|user_id=%s", endpoint, userID)
	val, err := rdb.Get(ctx, key).Int()
	if err == nil {
		t.Logf("Redis key format: %s = %d (using '|' separator)", key, val)
		assert.Greater(t, val, 0)
	}

	// 5. Test metrics
	t.Log("\n=== Testing metrics ===")

	metrics, err := getMetrics(operatorURL)
	require.NoError(t, err)
	t.Logf("Metrics sample:\n%s", metrics[:min(500, len(metrics))])

	assert.Contains(t, metrics, "ratelimit_violating_users_total")
	assert.Contains(t, metrics, "ratelimit_checks_total")

	// 6. Test reset
	t.Log("\n=== Testing rate limit reset ===")

	err = resetUserRateLimit(operatorURL, userID)
	require.NoError(t, err)
	t.Log("Rate limits reset")

	time.Sleep(2 * time.Second)

	// Request after reset should be allowed
	statusCode, err = sendGatewayRequest(gatewayURL, endpoint, jwtToken)
	require.NoError(t, err)
	t.Logf("Request after reset: HTTP %d", statusCode)
	assert.Equal(t, 200, statusCode)

	// 7. Performance test
	t.Log("\n=== Performance test ===")

	durations := make([]time.Duration, 0)
	for i := 0; i < 10; i++ {
		start := time.Now()
		_, err := sendGatewayRequest(gatewayURL, endpoint, jwtToken)
		duration := time.Since(start)
		durations = append(durations, duration)
		require.NoError(t, err)
	}

	// Calculate average latency
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	avgLatency := total / time.Duration(len(durations))

	t.Logf("Average latency over 10 requests: %v", avgLatency)
	t.Logf("Individual latencies: %v", durations)

	// 8. Concurrent requests test
	t.Log("\n=== Concurrent requests test ===")

	concurrency := 10
	results := make(chan int, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			code, err := sendGatewayRequest(gatewayURL, endpoint, jwtToken)
			if err != nil {
				results <- 0
				return
			}
			results <- code
		}()
	}

	allowed := 0
	rejected := 0
	for i := 0; i < concurrency; i++ {
		code := <-results
		if code == 200 {
			allowed++
		} else if code == 429 {
			rejected++
		}
	}

	t.Logf("Concurrent %d requests: %d allowed, %d rejected", concurrency, allowed, rejected)

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

func sendGatewayRequest(gatewayURL, endpoint, jwtToken string) (int, error) {
	url := gatewayURL + endpoint

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("x-user-id", "cloud-e2e-user")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func getMetrics(operatorURL string) (string, error) {
	resp, err := http.Get(operatorURL + "/metrics")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func resetUserRateLimit(operatorURL, userID string) error {
	url := fmt.Sprintf("%s/api/v1/users/%s/reset", operatorURL, userID)
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

func generateJWTToken(userID string) (string, error) {
	// Use pre-generated token or generate dynamically
	// For cloud E2E, use a known valid token
	return utils.GetEnv("TEST_JWT_TOKEN", ""), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
