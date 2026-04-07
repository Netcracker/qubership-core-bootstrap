package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"
	"net/http"

	"ratelimit-service/pkg/api"
	"ratelimit-service/pkg/controller"
	"ratelimit-service/pkg/metrics"
	ratelimitpkg "ratelimit-service/pkg/ratelimit"
	"ratelimit-service/pkg/utils"

	"github.com/redis/go-redis/v9"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := getKubeConfig()
	if err != nil {
		klog.Fatalf("Failed to get kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create kubernetes client: %v", err)
	}

	redisAddr := utils.GetEnv("REDIS_ADDR", "redis:6379")

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		DB:       0,
		Password: utils.GetEnv("REDIS_PASSWORD", ""),
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		klog.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer rdb.Close()

	metricsCollector := metrics.NewDefaultMetricsCollector()
	metrics.SetGlobalMetrics(metricsCollector)

	rateLimitMetrics := ratelimitpkg.NewMetrics(prometheus.DefaultRegisterer)
	limiter := ratelimitpkg.NewLimiter(rdb, rateLimitMetrics)
	rateLimitManager := ratelimitpkg.NewRateLimitManager(limiter)

	rateLimitManager.AddRule(&ratelimitpkg.Rule{
		Name:      "default",
		Pattern:   ".*",
		Limit:     60,
		Window:    time.Minute,
		Algorithm: ratelimitpkg.AlgorithmSlidingWindow,
	})

	rateLimitManager.AddRule(&ratelimitpkg.Rule{
		Name:      "api_strict",
		Pattern:   "/api/v1/.*",
		Limit:     10,
		Window:    time.Minute,
		Algorithm: ratelimitpkg.AlgorithmSlidingWindow,
	})

	rateLimitManager.AddRule(&ratelimitpkg.Rule{
		Name:      "burst_allowed",
		Pattern:   "/health",
		Limit:     100,
		Window:    time.Second,
		Algorithm: ratelimitpkg.AlgorithmTokenBucket,
	})

	klog.Info("Rate limiter initialized with rules")

	grpcPort := utils.GetEnv("GRPC_PORT", "8081")

	grpcServer, err := ratelimitpkg.StartGRPCServer(grpcPort, rateLimitManager)
	if err != nil {
		klog.Fatalf("Failed to start gRPC server: %v", err)
	}
	defer grpcServer.GracefulStop()

	redisClient, _ := ratelimitpkg.NewRedisClient(redisAddr, utils.GetEnv("REDIS_PASSWORD", ""), 0)

	controller := controller.NewConfigMapController(clientset, redisClient, rateLimitManager)

	go startMetricsServer(":9090", metricsCollector.GetRegistry())

	apiServer := api.NewServer(redisClient, controller, rateLimitManager)
	apiReady := make(chan struct{})
	go func() {
		close(apiReady)
		if err := apiServer.Run(":8082"); err != nil {
			klog.Errorf("API server error: %v", err)
			cancel()
		}
	}()

	<-apiReady
	time.Sleep(100 * time.Millisecond)

	metricsService := metrics.NewMetricsCollectorService(redisClient, metricsCollector, 30*time.Second)
	go metricsService.Start(ctx)

	klog.Info("RateLimit Operator started successfully")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	klog.Infof("Received signal: %v, starting graceful shutdown...", sig)

	shutdownTimeout := 30 * time.Second
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	done := make(chan struct{})

	go func() {
		defer close(done)

		klog.Info("Stopping API server...")
		apiServer.Stop()

		klog.Info("Stopping metrics service...")
		metricsService.Stop()

		klog.Info("Stopping controller...")
		controller.Stop()

		klog.Info("Closing Redis connections...")
		rdb.Close()
		redisClient.Close()

		klog.Info("All components stopped successfully")
	}()

	select {
	case <-done:
		klog.Info("Graceful shutdown completed")
	case <-shutdownCtx.Done():
		klog.Warning("Graceful shutdown timeout, forcing exit")
	}

	klog.Info("RateLimit Operator stopped")
}

func getKubeConfig() (*rest.Config, error) {
	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
	}
	return clientcmd.BuildConfigFromFlags("", utils.GetEnv("KUBECONFIG", ""))
}

func startMetricsServer(addr string, registry *prometheus.Registry) {
    mux := http.NewServeMux()
    mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{
        EnableOpenMetrics: true,
    }))
    
    server := &http.Server{
        Addr:    addr,
        Handler: mux,
    }
    
    klog.Infof("Metrics server listening on %s", addr)
    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        klog.Errorf("Metrics server error: %v", err)
    }
}