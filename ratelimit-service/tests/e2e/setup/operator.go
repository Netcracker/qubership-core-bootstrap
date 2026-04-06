package setup

import (
	"context"
	"ratelimit-service/pkg/api"
	"time"

	"ratelimit-service/pkg/controller"
	"ratelimit-service/pkg/metrics"
	"ratelimit-service/pkg/ratelimit"

	"github.com/redis/go-redis/v9"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

type LocalOperator struct {
	controller       *controller.ConfigMapController
	apiServer        *api.Server
	redisClient      *ratelimit.RedisClient
	limiter          *ratelimit.Limiter
	rateLimitManager *ratelimit.RateLimitManager
	metricsService   *metrics.MetricsCollectorService
	cancel           context.CancelFunc
}

func NewLocalOperator(kubeconfigPath string) (*LocalOperator, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	redisAddr := "localhost:6379"
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0,
	})

	metricsCollector := metrics.NewDefaultMetricsCollector()
	metrics.SetGlobalMetrics(metricsCollector)

	limiter := ratelimit.NewLimiter(rdb, metricsCollector)
	rateLimitManager := ratelimit.NewRateLimitManager(limiter)

	rateLimitManager.AddRule(&ratelimit.Rule{
		Name:      "api_strict",
		Pattern:   "/test",
		Limit:     2,
		Window:    10 * time.Second,
		Algorithm: ratelimit.AlgorithmFixedWindow,
	})

	redisClient, _ := ratelimit.NewRedisClient(redisAddr, "", 0)

	metricsService := metrics.NewMetricsCollectorService(redisClient, metricsCollector, 5*time.Second)

	controller := controller.NewConfigMapController(clientset, redisClient, rateLimitManager)
	apiServer := api.NewServer(redisClient, controller, rateLimitManager)

	return &LocalOperator{
		controller:       controller,
		apiServer:        apiServer,
		redisClient:      redisClient,
		limiter:          limiter,
		rateLimitManager: rateLimitManager,
		metricsService:   metricsService,
	}, nil
}

func (op *LocalOperator) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	op.cancel = cancel

	go op.metricsService.Start(ctx)

	go op.controller.Run(ctx)

	go func() {
		if err := op.apiServer.Run(":8082"); err != nil {
			klog.Errorf("API server error: %v", err)
		}
	}()

	klog.Info("Local operator started successfully")
	return nil
}

func (op *LocalOperator) Stop() {
	if op.cancel != nil {
		op.cancel()
	}
	op.metricsService.Stop()
	op.controller.Stop()
	op.redisClient.Close()
	klog.Info("Local operator stopped")
}
