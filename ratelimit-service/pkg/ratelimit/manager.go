package ratelimit

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

type RateLimitManager struct {
	limiter     *Limiter
	rules       sync.Map
	defaultRule *Rule
}

type Rule struct {
	Name      string
	Pattern   string
	Regex     *regexp.Regexp
	Limit     int
	Window    time.Duration
	Algorithm Algorithm
}

func NewRateLimitManager(limiter *Limiter) *RateLimitManager {
	if limiter == nil {
		return nil
	}
	return &RateLimitManager{
		limiter: limiter,
		defaultRule: &Rule{
			Name:      "default",
			Pattern:   ".*",
			Limit:     60,
			Window:    time.Minute,
			Algorithm: AlgorithmSlidingWindow,
		},
	}
}

func (m *RateLimitManager) AddRule(rule *Rule) {
	if m == nil {
		return
	}
	if rule.Pattern != "" {
		rule.Regex = regexp.MustCompile(rule.Pattern)
	}
	m.rules.Store(rule.Name, rule)
	klog.Infof("Added rate limit rule: %s (limit: %d/%s, pattern: %s, algorithm: %s)",
		rule.Name, rule.Limit, rule.Window, rule.Pattern, rule.Algorithm)
}

func (m *RateLimitManager) RemoveRule(name string) {
	if m == nil {
		return
	}
	m.rules.Delete(name)
	klog.Infof("Removed rate limit rule: %s", name)
}

func (m *RateLimitManager) GetRule(key string) *Rule {
	if m == nil {
		return nil
	}
	var matched *Rule
	m.rules.Range(func(k, v interface{}) bool {
		rule := v.(*Rule)
		if rule.Regex != nil && rule.Regex.MatchString(key) {
			matched = rule
			return false
		}
		return true
	})

	if matched == nil {
		return m.defaultRule
	}
	return matched
}

func (m *RateLimitManager) Check(ctx context.Context, key string) (*Result, error) {
	if m == nil || m.limiter == nil {
		return &Result{Allowed: true}, nil
	}
	rule := m.GetRule(key)
	if rule == nil {
		return &Result{Allowed: true}, nil
	}

	klog.V(4).Infof("Checking rate limit for key=%s with rule=%s, limit=%d, window=%v, algorithm=%s",
		key, rule.Name, rule.Limit, rule.Window, rule.Algorithm)

	return m.limiter.AllowWithAlgorithm(ctx, key, rule.Limit, rule.Window, rule.Algorithm)
}

func (m *RateLimitManager) CheckWithComponents(ctx context.Context, components map[string]string, separator string) (*Result, error) {
	if m == nil {
		return &Result{Allowed: true}, nil
	}
	key := buildKey(components, separator)
	return m.Check(ctx, key)
}

func (m *RateLimitManager) ClearRules() {
	if m == nil {
		return
	}
	m.rules = sync.Map{}
	klog.Info("All rate limit rules cleared")
}

func buildKey(components map[string]string, separator string) string {
	parts := make([]string, 0, len(components))
	for k, v := range components {
		value := strings.ReplaceAll(v, separator, "\\"+separator)
		parts = append(parts, k+"="+value)
	}
	return strings.Join(parts, separator)
}
