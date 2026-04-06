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
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_RateLimitThroughOperatorAPI(t *testing.T) {
	ctx := context.Background()

	// 1. Setup
	t.Log("=== Setting up test environment ===")

	kubeconfig := utils.GetEnv("KUBECONFIG", utils.GetEnv("HOME", "") + "/.kube/config")

	// Port-forward for Redis
	redisPF := setup.NewPortForward("core-1-core", "service/redis", "6379", "6379")
	require.NoError(t, redisPF.Start())
	defer redisPF.Stop()

	// Start local operator
	operator, err := setup.NewLocalOperator(kubeconfig)
	require.NoError(t, err)
	require.NoError(t, operator.Start(ctx))
	defer operator.Stop()

	time.Sleep(3 * time.Second)

	operatorURL := "http://localhost:8082"
	userID := "e2e-test-user"

	// 2. Configure rate limit via operator API
	t.Log("=== Configuring rate limit ===")

	rule := map[string]interface{}{
		"name":       "test_rule",
		"pattern":    ".*",
		"limit":      2,
		"window_sec": 10,
		"algorithm":  "fixed_window",
	}

	body, _ := json.Marshal(rule)
	resp, err := http.Post(operatorURL+"/api/v1/ratelimit/rules", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	resp.Body.Close()
	t.Log("Rate limit rule configured: 2 requests per 10 seconds")

	// 3. Test rate limiting through operator API
	t.Log("=== Testing rate limit ===")

	// First 2 requests should be allowed
	for i := 0; i < 2; i++ {
		allowed, current, err := checkRateLimit(operatorURL, "/test", userID)
		require.NoError(t, err)
		t.Logf("Request %d: allowed=%v, current=%d", i+1, allowed, current)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
	}

	// Third request should be rejected
	allowed, current, err := checkRateLimit(operatorURL, "/test", userID)
	require.NoError(t, err)
	t.Logf("Request 3: allowed=%v, current=%d", allowed, current)
	assert.False(t, allowed, "Third request should be rejected")

	// 4. Check Redis
	t.Log("\n=== Checking Redis ===")

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	defer rdb.Close()

	key := fmt.Sprintf("path=/test|user_id=%s", userID)
	val, err := rdb.Get(ctx, key).Int()
	if err == nil {
		t.Logf("Redis key %s = %d", key, val)
	}

	// 5. Get violating users
	t.Log("\n=== Getting Violating Users ===")

	violatingUsers, err := getViolatingUsers(operatorURL)
	require.NoError(t, err)
	t.Logf("Violating users: %v", violatingUsers)

	// 6. Reset rate limits
	t.Log("\n=== Resetting Rate Limits ===")

	err = resetUserRateLimit(operatorURL, userID)
	require.NoError(t, err)
	t.Log("✓ Rate limits reset successfully")

	// 7. Verify reset
	t.Log("\n=== Verifying Reset ===")

	allowed, current, err = checkRateLimit(operatorURL, "/test", userID)
	require.NoError(t, err)
	t.Logf("Request after reset: allowed=%v, current=%d", allowed, current)
	assert.True(t, allowed, "Request after reset should be allowed")

	// 8. Clean up
	t.Log("\n=== Cleaning up ===")

	req, err := http.NewRequest("DELETE", operatorURL+"/api/v1/ratelimit/rules/test_rule", nil)
	require.NoError(t, err)
	client := &http.Client{}
	resp, err = client.Do(req)
	if err == nil {
		resp.Body.Close()
		t.Log("Rate limit rule removed")
	}

	t.Log("\n=== ✅ Test completed successfully! ===")
}

// Helper functions
func checkRateLimit(apiURL, path, userID string) (bool, int, error) {
	reqBody := map[string]interface{}{
		"components": map[string]string{
			"path":    path,
			"user_id": userID,
		},
	}

	body, _ := json.Marshal(reqBody)
	resp, err := http.Post(apiURL+"/api/v1/ratelimit/check", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, 0, err
	}

	allowed, ok := result["allowed"].(bool)
	if !ok {
		return false, 0, fmt.Errorf("invalid response format")
	}

	current := 0
	if c, ok := result["current"].(float64); ok {
		current = int(c)
	}

	return allowed, current, nil
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
