package configserver

import (
	"context"
	"fmt"
	"github.com/netcracker/core-bootstrap/v2/scripts/consul"
	"github.com/netcracker/core-bootstrap/v2/utils"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
)

const (
	ConfigServerConsulTokenName = "config-server-consul-token"
)

var logger = logging.GetLogger("configserver")

type Configurer struct {
	Namespace        string
	consulConfigurer *consul.Configurer
	secretName       string
}

func New(consulConfigurer *consul.Configurer) *Configurer {
	return &Configurer{consulConfigurer: consulConfigurer}
}

func (c *Configurer) Configure(accessor func(string) string) error {
	c.Namespace = utils.MustGetEnv(accessor, "NAMESPACE")
	c.secretName = ConfigServerConsulTokenName

	return nil
}

func (c *Configurer) Execute(ctx context.Context) error {
	if c.consulConfigurer.Enabled {
		if err := c.configureConsulAccess(ctx); err != nil {
			return utils.LogError(logger, ctx, "error configure config-server access to consul: %w", err)
		}
	} else {
		logger.InfoC(ctx, "CONSUL_ENABLED is not set. Skip consul integration")
	}

	return nil
}

func (c *Configurer) configureConsulAccess(ctx context.Context) error {
	logger.InfoC(ctx, "*** Starting config_server_consul ***")

	policyConfigWrite := consul.Policy{
		Name:        fmt.Sprintf("%s_config-edit", c.Namespace),
		Description: "Policy for configs write",
		Rules:       fmt.Sprintf(`key_prefix "config/%s/" { policy = "write" }`, c.Namespace),
	}

	policyLoggingRead := consul.Policy{
		Name:        fmt.Sprintf("%s_config-server_logging", c.Namespace),
		Description: "Policy for logging read",
		Rules:       fmt.Sprintf(`key_prefix "logging/%s/config-server" { policy = "read" }`, c.Namespace),
	}

	requiredPolicies := []consul.Policy{policyLoggingRead, policyConfigWrite}

	err := c.consulConfigurer.CheckAndCreateConsulPoliciesAndToken(ctx, c.secretName, requiredPolicies, policyConfigWrite.Name)
	if err != nil {
		return utils.LogError(logger, ctx, "error CheckAndCreateConsulPoliciesAndToken for config server: %w", err)
	}

	logger.InfoC(ctx, "### Finished config_server_consul ***")
	return nil
}
