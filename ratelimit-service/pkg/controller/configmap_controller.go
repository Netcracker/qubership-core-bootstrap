package controller

import (
    "context"
    "fmt"
    "sync"
    "time"

    "ratelimit-service/pkg/ratelimit"
    "ratelimit-service/pkg/utils"

    v1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/watch"
    "k8s.io/client-go/kubernetes"
    "k8s.io/klog/v2"
)

type ConfigMapController struct {
    clientset        kubernetes.Interface
    redisClient      *ratelimit.RedisClient
    rateLimitManager *ratelimit.RateLimitManager
    configs          sync.Map
    namespace        string
    stopChan         chan struct{}
    wg               sync.WaitGroup
}

func NewConfigMapController(clientset kubernetes.Interface, redisClient *ratelimit.RedisClient, rateLimitManager *ratelimit.RateLimitManager) *ConfigMapController {
    namespace := utils.GetEnv("NAMESPACE", "core-1-core")

    return &ConfigMapController{
        clientset:        clientset,
        redisClient:      redisClient,
        rateLimitManager: rateLimitManager,
        namespace:        namespace,
        stopChan:         make(chan struct{}),
    }
}

func (c *ConfigMapController) Run(ctx context.Context) {
    klog.Info("Starting ConfigMap controller")

    if err := c.loadExistingConfigs(ctx); err != nil {
        klog.Errorf("Failed to load existing configs: %v", err)
    }

    c.wg.Add(1)
    go c.watchWithReconnect(ctx)

    <-c.stopChan
    c.wg.Wait()
    klog.Info("ConfigMap controller stopped")
}

func (c *ConfigMapController) loadExistingConfigs(ctx context.Context) error {
    cms, err := c.clientset.CoreV1().ConfigMaps(c.namespace).List(ctx, metav1.ListOptions{
        LabelSelector: "rate-limit-config=true",
    })
    if err != nil {
        return fmt.Errorf("failed to list configmaps: %w", err)
    }

    klog.Infof("Found %d existing ConfigMaps with rate-limit-config label", len(cms.Items))

    for _, cm := range cms.Items {
        if err := c.processConfigMap(ctx, &cm); err != nil {
            klog.Errorf("Failed to process ConfigMap %s: %v", cm.Name, err)
        }
    }

    return nil
}

func (c *ConfigMapController) watchWithReconnect(ctx context.Context) {
    defer c.wg.Done()

    for {
        select {
        case <-ctx.Done():
            return
        case <-c.stopChan:
            return
        default:
            c.watchConfigMaps(ctx)
            klog.Warning("ConfigMap watcher disconnected, reconnecting in 5 seconds...")
            time.Sleep(5 * time.Second)
        }
    }
}

func (c *ConfigMapController) watchConfigMaps(ctx context.Context) {
    watcher, err := c.clientset.CoreV1().ConfigMaps(c.namespace).Watch(ctx, metav1.ListOptions{
        LabelSelector: "rate-limit-config=true",
    })
    if err != nil {
        klog.Errorf("Failed to create watcher: %v", err)
        return
    }
    defer watcher.Stop()

    klog.Info("ConfigMap watcher started")

    for {
        select {
        case <-ctx.Done():
            return
        case <-c.stopChan:
            return
        case event, ok := <-watcher.ResultChan():
            if !ok {
                return
            }

            cm, ok := event.Object.(*v1.ConfigMap)
            if !ok {
                klog.Warning("Unexpected object type in watcher")
                continue
            }

            switch event.Type {
            case watch.Added, watch.Modified:
                klog.Infof("ConfigMap %s: %s", cm.Name, event.Type)
                if err := c.processConfigMap(ctx, cm); err != nil {
                    klog.Errorf("Failed to process ConfigMap %s: %v", cm.Name, err)
                }
            case watch.Deleted:
                klog.Infof("ConfigMap deleted: %s", cm.Name)
                c.deleteConfig(cm.Name)
            }
        }
    }
}

func (c *ConfigMapController) processConfigMap(ctx context.Context, cm *v1.ConfigMap) error {
    configData, ok := cm.Data["config.yaml"]
    if !ok {
        configData, ok = cm.Data["app-config.yaml"]
        if !ok {
            return fmt.Errorf("no config.yaml or app-config.yaml found in ConfigMap %s", cm.Name)
        }
    }

    rules, err := c.parseConfig(configData)
    if err != nil {
        return fmt.Errorf("failed to parse config: %w", err)
    }

    c.configs.Store(cm.Name, rules)

    for _, rule := range rules {
        c.rateLimitManager.AddRule(rule)
    }

    klog.Infof("Processed ConfigMap %s with %d rules", cm.Name, len(rules))
    return nil
}

func (c *ConfigMapController) parseConfig(configData string) ([]*ratelimit.Rule, error) {

    var rules []*ratelimit.Rule

    rules = append(rules, &ratelimit.Rule{
        Name:      "default",
        Pattern:   ".*",
        Limit:     60,
        Window:    time.Minute,
        Algorithm: ratelimit.AlgorithmSlidingWindow,
    })

    return rules, nil
}

func (c *ConfigMapController) deleteConfig(name string) {
    c.configs.Delete(name)
    klog.Infof("ConfigMap %s removed", name)
}

func (c *ConfigMapController) ReloadConfig(ctx context.Context) error {
    klog.Info("Manual config reload triggered")

    // c.rateLimitManager.ClearRules()

    if err := c.loadExistingConfigs(ctx); err != nil {
        return fmt.Errorf("failed to reload configs: %w", err)
    }

    klog.Info("Config reload completed")
    return nil
}

func (c *ConfigMapController) getConfigCount() int {
    count := 0
    c.configs.Range(func(key, value interface{}) bool {
        count++
        return true
    })
    return count
}

func (c *ConfigMapController) Stop() {
    klog.Info("Stopping ConfigMap controller...")
    close(c.stopChan)
    c.wg.Wait()
    klog.Info("ConfigMap controller stopped")
}
