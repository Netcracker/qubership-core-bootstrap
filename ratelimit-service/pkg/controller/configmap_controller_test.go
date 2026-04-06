package controller

import (
	"context"
	"testing"
	"time"

	"ratelimit-service/pkg/ratelimit"
	"ratelimit-service/pkg/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var namespace = utils.GetEnv("NAMESPACE", "core-1-core")

func setupTestController(t *testing.T) (*ConfigMapController, *miniredis.Miniredis, func()) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	metrics := ratelimit.NewMetrics(nil)
	limiter := ratelimit.NewLimiter(rdb, metrics)
	manager := ratelimit.NewRateLimitManager(limiter)

	clientset := fake.NewSimpleClientset()
	controller := NewConfigMapController(clientset, nil, manager)

	cleanup := func() {
		rdb.Close()
		mr.Close()
	}

	return controller, mr, cleanup
}

func TestNewConfigMapController(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	manager := &ratelimit.RateLimitManager{}

	controller := NewConfigMapController(clientset, nil, manager)

	assert.NotNil(t, controller)
	assert.Equal(t, namespace, controller.namespace)
	assert.NotNil(t, controller.stopChan)
}

func TestReloadConfig(t *testing.T) {
	controller, _, cleanup := setupTestController(t)
	defer cleanup()

	ctx := context.Background()

	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: namespace,
			Labels: map[string]string{
				"rate-limit-config": "true",
			},
		},
		Data: map[string]string{
			"config.yaml": "test config data",
		},
	}

	_, err := controller.clientset.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	require.NoError(t, err)

	err = controller.ReloadConfig(ctx)
	assert.NoError(t, err)
}

func TestProcessConfigMap(t *testing.T) {
	controller, _, cleanup := setupTestController(t)
	defer cleanup()

	ctx := context.Background()

	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: namespace,
		},
		Data: map[string]string{
			"config.yaml": "test config data",
		},
	}

	err := controller.processConfigMap(ctx, cm)
	assert.NoError(t, err)

	value, ok := controller.configs.Load("test-config")
	assert.True(t, ok)
	assert.NotNil(t, value)
}

func TestDeleteConfig(t *testing.T) {
	controller, _, cleanup := setupTestController(t)
	defer cleanup()

	controller.configs.Store("test-config", "test data")
	assert.Equal(t, 1, controller.getConfigCount())

	controller.deleteConfig("test-config")
	assert.Equal(t, 0, controller.getConfigCount())
}

func TestGetConfigCount(t *testing.T) {
	controller, _, cleanup := setupTestController(t)
	defer cleanup()

	assert.Equal(t, 0, controller.getConfigCount())

	controller.configs.Store("config1", "data1")
	controller.configs.Store("config2", "data2")

	assert.Equal(t, 2, controller.getConfigCount())
}

func TestStop(t *testing.T) {
	controller, _, cleanup := setupTestController(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		controller.Run(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	controller.Stop()
	cancel()

	select {
	case <-controller.stopChan:
		// OK
	default:
		t.Error("stopChan should be closed")
	}
}
