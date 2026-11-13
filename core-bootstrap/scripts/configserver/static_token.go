package configserver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/netcracker/core-bootstrap/v2/utils"
	consulauth "github.com/netcracker/qubership-core-lib-go-rest-utils/v2/consul-propertysource"
)

const consulACLDisabledMessage = "ACL support disabled"

// staticTokenNeeded reports whether config-server needs the static Consul token: always in legacy, never in k8s,
// and in hybrid when it cannot log in to Consul with its ServiceAccount.
func (c *Configurer) staticTokenNeeded(ctx context.Context) (bool, error) {
	switch c.authSettings.Mode {
	case consulauth.AuthModeLegacy:
		return true, nil
	case consulauth.AuthModeK8s:
		return false, nil
	case consulauth.AuthModeHybrid:
		loginWorks, err := c.serviceAccountLoginWorks(ctx)
		return !loginWorks, err
	default:
		return false, fmt.Errorf("unknown Consul auth mode '%s'", c.authSettings.Mode)
	}
}

// serviceAccountLoginWorks reads the auth method with the admin token and reports whether a ServiceAccount login can
// succeed: Consul has the method, or has ACLs disabled and needs no login. A missing method means it cannot; any other
// answer is an error.
func (c *Configurer) serviceAccountLoginWorks(ctx context.Context) (bool, error) {
	methodURL := fmt.Sprintf("%s/v1/acl/auth-method/%s", c.consulConfigurer.Address, url.PathEscape(c.authSettings.AuthMethod))
	resp, err := utils.RestyClient.R().
		SetContext(ctx).
		SetHeader("X-Consul-Token", c.consulConfigurer.AdminToken()).
		Get(methodURL)
	if err != nil {
		return false, fmt.Errorf("failed to read Consul auth method '%s': %w", c.authSettings.AuthMethod, err)
	}

	switch {
	case resp.StatusCode() == http.StatusOK:
		logger.InfoC(ctx, "Consul has auth method '%s': issuing no token", c.authSettings.AuthMethod)
		return true, nil
	case resp.StatusCode() == http.StatusNotFound:
		logger.InfoC(ctx, "Consul has no auth method '%s': issuing the static token", c.authSettings.AuthMethod)
		return false, nil
	case resp.StatusCode() == http.StatusUnauthorized && resp.String() == consulACLDisabledMessage:
		logger.InfoC(ctx, "Consul ACLs are disabled: issuing no token")
		return true, nil
	default:
		return false, fmt.Errorf("failed to read Consul auth method '%s': unexpected status code %d, body: %s",
			c.authSettings.AuthMethod, resp.StatusCode(), resp.String())
	}
}
