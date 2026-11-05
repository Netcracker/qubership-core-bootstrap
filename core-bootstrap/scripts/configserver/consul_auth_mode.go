package configserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/netcracker/core-bootstrap/v2/utils"
)

type consulAuthMode string

const (
	consulAuthModeLegacy consulAuthMode = "legacy"
	consulAuthModeHybrid consulAuthMode = "hybrid"
	consulAuthModeK8s    consulAuthMode = "k8s"

	defaultConsulAuthMethod   = "applications-k8s-m2m"
	defaultConsulAuthAudience = "netcracker"

	consulACLDisabledMessage = "ACL support disabled"
)

func readConsulAuthMode(accessor func(string) string) (consulAuthMode, error) {
	source := "CONSUL_AUTH_MODE"
	value := strings.TrimSpace(accessor(source))
	if value == "" {
		source = "M2M_AUTH_MODE"
		value = strings.ToLower(strings.TrimSpace(accessor(source)))
	}
	switch mode := consulAuthMode(value); mode {
	case "":
		return consulAuthModeLegacy, nil
	case consulAuthModeLegacy, consulAuthModeHybrid, consulAuthModeK8s:
		return mode, nil
	default:
		return "", fmt.Errorf("%s has unsupported value %q: set it to legacy, hybrid, or k8s", source, value)
	}
}

func valueOrDefault(value, defaultValue string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return defaultValue
}

type consulAuthMethod struct {
	Config struct {
		BoundAudiences []string `json:"BoundAudiences"`
	} `json:"Config"`
}

func (c *Configurer) resolveConsulAuthMode(ctx context.Context) (consulAuthMode, error) {
	if c.authMode != consulAuthModeHybrid {
		return c.authMode, nil
	}

	methodURL := fmt.Sprintf("%s/v1/acl/auth-method/%s", c.consulConfigurer.Address, url.PathEscape(c.authMethod))
	resp, err := utils.RestyClient.R().
		SetContext(ctx).
		SetHeader("X-Consul-Token", c.adminToken).
		Get(methodURL)
	if err != nil {
		return "", fmt.Errorf("failed to read Consul auth method '%s': %w", c.authMethod, err)
	}

	switch {
	case resp.StatusCode() == http.StatusNotFound:
		logger.InfoC(ctx, "Consul has no auth method '%s': issuing the static token", c.authMethod)
		return consulAuthModeLegacy, nil
	case resp.StatusCode() == http.StatusUnauthorized && strings.Contains(resp.String(), consulACLDisabledMessage):
		logger.InfoC(ctx, "Consul ACLs are disabled: issuing no token")
		return consulAuthModeK8s, nil
	case resp.StatusCode() != http.StatusOK:
		return "", fmt.Errorf("failed to read Consul auth method '%s': unexpected status code %d, body: %s",
			c.authMethod, resp.StatusCode(), resp.String())
	}

	var method consulAuthMethod
	if err := json.Unmarshal(resp.Body(), &method); err != nil {
		return "", fmt.Errorf("failed to decode Consul auth method '%s': %w", c.authMethod, err)
	}
	if !slices.Contains(method.Config.BoundAudiences, c.authAudience) {
		logger.InfoC(ctx, "Consul auth method '%s' does not accept audience '%s': issuing the static token",
			c.authMethod, c.authAudience)
		return consulAuthModeLegacy, nil
	}
	logger.InfoC(ctx, "Consul auth method '%s' accepts audience '%s': issuing no token", c.authMethod, c.authAudience)
	return consulAuthModeK8s, nil
}
