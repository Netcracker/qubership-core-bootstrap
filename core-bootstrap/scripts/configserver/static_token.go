package configserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/netcracker/core-bootstrap/v2/utils"
	consulauth "github.com/netcracker/qubership-core-lib-go-rest-utils/v2/consul-propertysource"
)

const consulACLDisabledMessage = "ACL support disabled"

type consulAuthMethod struct {
	Config struct {
		BoundAudiences []string `json:"BoundAudiences"`
	} `json:"Config"`
}

func (c *Configurer) staticTokenNeeded(ctx context.Context) (bool, error) {
	switch c.authSettings.Mode {
	case consulauth.AuthModeLegacy:
		return true, nil
	case consulauth.AuthModeK8s:
		return false, nil
	}

	methodURL := fmt.Sprintf("%s/v1/acl/auth-method/%s", c.consulConfigurer.Address, url.PathEscape(c.authSettings.AuthMethod))
	resp, err := utils.RestyClient.R().
		SetContext(ctx).
		SetHeader("X-Consul-Token", c.consulConfigurer.AdminToken()).
		Get(methodURL)
	if err != nil {
		return false, fmt.Errorf("failed to read Consul auth method '%s': %w", c.authSettings.AuthMethod, err)
	}

	switch {
	case resp.StatusCode() == http.StatusNotFound:
		logger.InfoC(ctx, "Consul has no auth method '%s': issuing the static token", c.authSettings.AuthMethod)
		return true, nil
	case resp.StatusCode() == http.StatusUnauthorized && resp.String() == consulACLDisabledMessage:
		logger.InfoC(ctx, "Consul ACLs are disabled: issuing no token")
		return false, nil
	case resp.StatusCode() != http.StatusOK:
		return false, fmt.Errorf("failed to read Consul auth method '%s': unexpected status code %d, body: %s",
			c.authSettings.AuthMethod, resp.StatusCode(), resp.String())
	}

	var method consulAuthMethod
	if err := json.Unmarshal(resp.Body(), &method); err != nil {
		return false, fmt.Errorf("failed to decode Consul auth method '%s': %w", c.authSettings.AuthMethod, err)
	}
	if !slices.Contains(method.Config.BoundAudiences, c.authSettings.Audience) {
		logger.InfoC(ctx, "Consul auth method '%s' does not accept audience '%s': issuing the static token",
			c.authSettings.AuthMethod, c.authSettings.Audience)
		return true, nil
	}
	logger.InfoC(ctx, "Consul auth method '%s' accepts audience '%s': issuing no token", c.authSettings.AuthMethod, c.authSettings.Audience)
	return false, nil
}
