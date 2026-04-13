package main

import (
    "context"
    "flag"
    "os"
    "os/signal"
    "syscall"
    "time"

    "ratelimit-service/pkg/api"
    "ratelimit-service/pkg/controller"
    "ratelimit-service/pkg/metrics"
    "ratelimit-service/pkg/ratelimit"
    "ratelimit-service/pkg/utils"

    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/klog/v2"
)

func main() {
    klog.InitFlags(nil)
    flag.Parse()

    redisAddr := utils.GetEnv("REDIS_ADDR", "localhost:6379")
    apiPort := utils.GetEnv("API_PORT", "8082")
    grpcPort := utils.GetEnv("GRPC_PORT", "8081")
    namespace := utils.GetEnv("NAMESPACE", "core-1-core")

    // Create Redis client
    redisClient, err := ratelimit.NewRedisClient(redisAddr, "", 0)
    if err != nil {
        klog.Fatalf("Failed to create Redis client: %v", err)
    }
    defer redisClient.Close()

    // Create rate limit manager
    rateLimitManager := ratelimit.NewRateLimitManager(redisClient)
    redisClient.SetManager(rateLimitManager)

    // Add default rule
    defaultRule := &ratelimit.Rule{
        Name:      "default",
        Pattern:   ".*",
        Limit:     60,
        Window:    time.Minute,
        Algorithm: ratelimit.FixedWindow,
        Priority:  0,
    }

    if err := rateLimitManager.AddRule(defaultRule); err != nil {
        klog.Warningf("Failed to add default rule: %v", err)
    }

    // Create Kubernetes client
    var clientset *kubernetes.Clientset
    var config *rest.Config
    
    // Try in-cluster config first (when running in Kubernetes)
    config, err = rest.InClusterConfig()
    if err != nil {
        // Fall back to kubeconfig for local development
        klog.Warningf("Failed to get in-cluster config: %v, falling back to kubeconfig", err)
        
        kubeconfig := utils.GetEnv("KUBECONFIG", utils.GetEnv("HOME", "")+"/.kube/config")
        config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
        if err != nil {
            klog.Fatalf("Failed to build kubeconfig: %v", err)
        }
    }
    
    clientset, err = kubernetes.NewForConfig(config)
    if err != nil {
        klog.Fatalf("Failed to create k8s client: %v", err)
    }
    
    klog.Info("Kubernetes client created successfully")
    klog.Infof("Using namespace: %s", namespace)

    // Create metrics collector
    metricsCollector := metrics.NewDefaultMetricsCollector()
    metrics.SetGlobalMetrics(metricsCollector)
    metricsService := metrics.NewMetricsCollectorService(redisClient, metricsCollector, 30*time.Second)

    // Create controller with namespace
    configMapController := controller.NewConfigMapController(clientset, redisClient, rateLimitManager)

    // Create API server
    apiServer := api.NewServer(redisClient, configMapController, rateLimitManager)

    // Start gRPC server for Envoy integration
    grpcServer, err := ratelimit.StartGRPCServer(grpcPort, rateLimitManager)
    if err != nil {
        klog.Fatalf("Failed to start gRPC server: %v", err)
    }
    defer grpcServer.GracefulStop()

    // Start services
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go metricsService.Start(ctx)
    go configMapController.Run(ctx)
    go func() {
        if err := apiServer.Run(":" + apiPort); err != nil {
            klog.Errorf("API server error: %v", err)
        }
    }()

    klog.Infof("All services started:")
    klog.Infof("  - HTTP API: port %s", apiPort)
    klog.Infof("  - gRPC API: port %s (for Envoy integration)", grpcPort)
    klog.Infof("  - Redis: %s", redisAddr)
    klog.Infof("  - Metrics collector: running")
    klog.Infof("  - ConfigMap controller: enabled (watching %s namespace)", namespace)

    // Wait for shutdown signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh

    klog.Info("Shutting down...")
    cancel()
    time.Sleep(2 * time.Second)
    klog.Info("Shutdown complete")
}