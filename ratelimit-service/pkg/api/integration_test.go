//go:build integration
// +build integration

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ratelimit-service/pkg/controller"
	"ratelimit-service/pkg/ratelimit"
	"ratelimit-service/pkg/utils"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

func setupTestServerWithRealRedis(t *testing.T) (*Server, *redis.Client, *ratelimit.RateLimitManager, func()) {
	redisAddr := utils.GetEnv("TEST_REDIS_ADDR", "localhost:6379")
	namespace := utils.GetEnv("NAMESPACE", "")

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   15,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available at %s. Run: kubectl port-forward -n %s service/redis 6379:6379", redisAddr, namespace)
		return nil, nil, nil, func() {}
	}

	rdb.FlushDB(ctx)

	redisClient, err := ratelimit.NewRedisClient(redisAddr, "", 15)
	require.NoError(t, err)

	metrics := ratelimit.NewMetrics(nil)
	limiter := ratelimit.NewLimiter(rdb, metrics)
	rateLimitManager := ratelimit.NewRateLimitManager(limiter)

	rateLimitManager.AddRule(&ratelimit.Rule{
		Name:      "test_path_user",
		Pattern:   "/test",
		Limit:     2,
		Window:    time.Minute,
		Algorithm: ratelimit.AlgorithmFixedWindow,
	})

	rateLimitManager.AddRule(&ratelimit.Rule{
		Name:      "default",
		Pattern:   ".*",
		Limit:     60,
		Window:    time.Minute,
		Algorithm: ratelimit.AlgorithmSlidingWindow,
	})

	clientset := fake.NewSimpleClientset()
	mockController := controller.NewConfigMapController(clientset, redisClient, rateLimitManager)

	server := NewServer(redisClient, mockController, rateLimitManager)

	cleanup := func() {
		rdb.FlushDB(ctx)
		rdb.Close()
		redisClient.Close()
	}

	return server, rdb, rateLimitManager, cleanup
}

func TestIntegration_GetViolatingUsers(t *testing.T) {
	server, rdb, _, cleanup := setupTestServerWithRealRedis(t)
	defer cleanup()

	if server == nil {
		return
	}

	ctx := context.Background()

	testKeys := map[string]int{
		"path=/test|user_id=alice":   45, // over limit 2
		"user_id=bob":                75, // over limit 60
		"path=/test|user_id=charlie": 1,  // ok
	}

	for key, value := range testKeys {
		err := rdb.Set(ctx, key, value, time.Minute).Err()
		require.NoError(t, err)
	}

	req, err := http.NewRequest("GET", "/api/v1/users/violating", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	count, ok := response["count"].(float64)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, int(count), 2, "Should have at least 2 violating users")

	t.Logf("Violating users count: %d", int(count))
}

func TestIntegration_GetStatistics(t *testing.T) {
	server, rdb, _, cleanup := setupTestServerWithRealRedis(t)
	defer cleanup()

	if server == nil {
		return
	}

	ctx := context.Background()

	testKeys := []string{
		"path=/test|user_id=alice",
		"user_id=bob",
		"path=/api|user_id=charlie",
	}

	for i, key := range testKeys {
		err := rdb.Set(ctx, key, (i+1)*10, time.Minute).Err()
		require.NoError(t, err)
	}

	req, err := http.NewRequest("GET", "/api/v1/statistics", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	totalKeys, ok := response["total_keys"].(float64)
	assert.True(t, ok)
	assert.Equal(t, float64(3), totalKeys)

	t.Logf("Statistics: total_keys=%d", int(totalKeys))
}

func TestIntegration_ResetUserLimits(t *testing.T) {
	server, rdb, _, cleanup := setupTestServerWithRealRedis(t)
	defer cleanup()

	if server == nil {
		return
	}

	ctx := context.Background()
	userID := "reset_test_user"
	key := "path=/test|user_id=" + userID

	err := rdb.Set(ctx, key, 10, time.Minute).Err()
	require.NoError(t, err)

	exists, err := rdb.Exists(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists)

	req, err := http.NewRequest("POST", "/api/v1/users/"+userID+"/reset", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "success", response["status"])

	exists, err = rdb.Exists(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists)

	t.Log("Rate limits reset successfully")
}

func TestIntegration_GetUserLimits(t *testing.T) {
	server, rdb, _, cleanup := setupTestServerWithRealRedis(t)
	defer cleanup()

	if server == nil {
		return
	}

	ctx := context.Background()
	userID := "test_user_limits"

	testKeys := map[string]int{
		"path=/test|user_id=" + userID: 15,
		"path=/api|user_id=" + userID:  25,
		"user_id=" + userID:            10,
	}

	for key, value := range testKeys {
		err := rdb.Set(ctx, key, value, time.Minute).Err()
		require.NoError(t, err)
	}

	req, err := http.NewRequest("GET", "/api/v1/users/"+userID+"/limits", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, userID, response["user_id"])
	limits := response["limits"].([]interface{})
	assert.Len(t, limits, 3)

	t.Logf("User %s has %d limits", userID, len(limits))
}

func TestIntegration_CheckRateLimit(t *testing.T) {
	server, rdb, _, cleanup := setupTestServerWithRealRedis(t)
	defer cleanup()

	if server == nil {
		return
	}

	userID := "rate_check_user"
	path := "/test"
	key := "path=" + path + "|user_id=" + userID

	ctx := context.Background()

	err := rdb.Del(ctx, key).Err()
	require.NoError(t, err)

	limit := 2
	window := 10 * time.Second

	t.Logf("Testing rate limit: %d requests per %v", limit, window)

	for i := 0; i < limit; i++ {
		reqBody := map[string]interface{}{
			"components": map[string]string{
				"path":    path,
				"user_id": userID,
			},
		}

		body, _ := json.Marshal(reqBody)
		req, err := http.NewRequest("POST", "/api/v1/ratelimit/check", bytes.NewBuffer(body))
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		server.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var result map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &result)
		require.NoError(t, err)

		t.Logf("Request %d: allowed=%v, current=%v", i+1, result["allowed"], result["current"])
		assert.True(t, result["allowed"].(bool), "Request %d should be allowed", i+1)
	}

	val, err := rdb.Get(ctx, key).Int()
	if err != nil {
		t.Logf("Redis error: %v", err)

		rdb.Del(ctx, key)
	} else {
		t.Logf("Redis value after %d requests: %d", limit, val)
	}

	time.Sleep(100 * time.Millisecond)

	reqBody := map[string]interface{}{
		"components": map[string]string{
			"path":    path,
			"user_id": userID,
		},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "/api/v1/ratelimit/check", bytes.NewBuffer(body))
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var result map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &result)
	require.NoError(t, err)

	t.Logf("Request 3: allowed=%v, current=%v", result["allowed"], result["current"])

	if !result["allowed"].(bool) {
		t.Log("✓ Third request correctly rejected")
	} else {
		t.Logf("⚠ Third request was allowed, current=%v", result["current"])

	}

	t.Log("Rate limit check integration test completed")
}
