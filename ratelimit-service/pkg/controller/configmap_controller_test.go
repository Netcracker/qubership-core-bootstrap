// pkg/controller/configmap_controller_test.go
package controller

import (
    "context"
    "testing"

    "ratelimit-service/pkg/ratelimit"
    "github.com/stretchr/testify/assert"
    "k8s.io/client-go/kubernetes/fake"
)

func TestConfigMapController_New(t *testing.T) {
    clientset := fake.NewSimpleClientset()
    
    redisClient, err := ratelimit.NewRedisClient("localhost:6379", "", 0)
    if err != nil {
        t.Skip("Redis not available, skipping test")
    }
    
    rateLimitManager := ratelimit.NewRateLimitManager(redisClient)
    redisClient.SetManager(rateLimitManager)
    
    controller := NewConfigMapController(clientset, redisClient, rateLimitManager)
    
    assert.NotNil(t, controller)
    assert.Equal(t, clientset, controller.clientset)
}

func TestConfigMapController_ParseConfig(t *testing.T) {
    clientset := fake.NewSimpleClientset()
    
    redisClient, err := ratelimit.NewRedisClient("localhost:6379", "", 0)
    if err != nil {
        t.Skip("Redis not available, skipping test")
    }
    
    rateLimitManager := ratelimit.NewRateLimitManager(redisClient)
    redisClient.SetManager(rateLimitManager)
    
    controller := NewConfigMapController(clientset, redisClient, rateLimitManager)
    
    configData := `
rules:
  - name: test_rule
    pattern: ".*user_id=test.*"
    limit: 10
    window_sec: 60
`
    rules, err := controller.parseConfig(configData)
    assert.NoError(t, err)
    assert.NotNil(t, rules)
}

func TestConfigMapController_Run(t *testing.T) {
    clientset := fake.NewSimpleClientset()
    
    redisClient, err := ratelimit.NewRedisClient("localhost:6379", "", 0)
    if err != nil {
        t.Skip("Redis not available, skipping test")
    }
    
    rateLimitManager := ratelimit.NewRateLimitManager(redisClient)
    redisClient.SetManager(rateLimitManager)
    
    controller := NewConfigMapController(clientset, redisClient, rateLimitManager)
    
    ctx, cancel := context.WithCancel(context.Background())
    
    go controller.Run(ctx)
    
    cancel()
    controller.Stop()
    
    assert.NotNil(t, controller)
}