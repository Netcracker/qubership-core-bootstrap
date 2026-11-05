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
	consulauth "github.com/netcracker/qubership-core-lib-go-rest-utils/v2/consul-propertysource"
)

const consulACLDisabledMessage = "ACL support disabled"

type consulAuthMethod struct {
	Config struct {
		BoundAudiences []string `json:"BoundAudiences"`
	} `json:"Config"`
}

func (c *Configurer) resolveConsulAuthMode(ctx context.Context) (consulauth.AuthMode, error) {
	if c.auth.Mode != consulauth.AuthModeHybrid {
		return c.auth.Mode, nil
	}

	methodURL := fmt.Sprintf("%s/v1/acl/auth-method/%s", c.consulConfigurer.Address, url.PathEscape(c.auth.AuthMethod))
	resp, err := utils.RestyClient.R().
		SetContext(ctx).
		SetHeader("X-Consul-Token", c.adminToken).
		Get(methodURL)
	if err != nil {
		return "", fmt.Errorf("failed to read Consul auth method '%s': %w", c.auth.AuthMethod, err)
	}

	switch {
	case resp.StatusCode() == http.StatusNotFound:
		logger.InfoC(ctx, "Consul has no auth method '%s': issuing the static token", c.auth.AuthMethod)
		return consulauth.AuthModeLegacy, nil
	case resp.StatusCode() == http.StatusUnauthorized && strings.TrimSpace(resp.String()) == consulACLDisabledMessage:
		logger.InfoC(ctx, "Consul ACLs are disabled: issuing no token")
		return consulauth.AuthModeK8s, nil
	case resp.StatusCode() != http.StatusOK:
		return "", fmt.Errorf("failed to read Consul auth method '%s': unexpected status code %d, body: %s",
			c.auth.AuthMethod, resp.StatusCode(), resp.String())
	}

	var method consulAuthMethod
	if err := json.Unmarshal(resp.Body(), &method); err != nil {
		return "", fmt.Errorf("failed to decode Consul auth method '%s': %w", c.auth.AuthMethod, err)
	}
	if !slices.Contains(method.Config.BoundAudiences, c.auth.Audience) {
		logger.InfoC(ctx, "Consul auth method '%s' does not accept audience '%s': issuing the static token",
			c.auth.AuthMethod, c.auth.Audience)
		return consulauth.AuthModeLegacy, nil
	}
	logger.InfoC(ctx, "Consul auth method '%s' accepts audience '%s': issuing no token", c.auth.AuthMethod, c.auth.Audience)
	return consulauth.AuthModeK8s, nil
}
