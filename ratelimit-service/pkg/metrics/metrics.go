package metrics

import (
    "net/http"
	
	"k8s.io/klog/v2"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	
)    

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