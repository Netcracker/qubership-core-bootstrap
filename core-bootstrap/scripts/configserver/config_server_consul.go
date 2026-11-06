package configserver

import (
	"context"
	"fmt"
	"github.com/netcracker/core-bootstrap/v2/scripts/consul"
	"github.com/netcracker/core-bootstrap/v2/utils"
	consulauth "github.com/netcracker/qubership-core-lib-go-rest-utils/v2/consul-propertysource"
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
	adminToken       string
	auth             consulauth.AuthSettings
	issueToken       func(ctx context.Context) error
	deleteSecret     func(ctx context.Context, namespace string, secretName string) error
}

func New(consulConfigurer *consul.Configurer) *Configurer {
	c := &Configurer{consulConfigurer: consulConfigurer, deleteSecret: utils.DeleteK8sSecret}
	c.issueToken = c.configureConsulAccess
	return c
}

func (c *Configurer) Configure(accessor func(string) string) error {
	c.Namespace = utils.MustGetEnv(accessor, "NAMESPACE")
	c.secretName = ConfigServerConsulTokenName
	c.adminToken = accessor("CONSUL_ADMIN_TOKEN")

	auth, err := consulauth.ReadAuthSettings()
	if err != nil {
		return err
	}
	c.auth = auth
	return nil
}

func (c *Configurer) Execute(ctx context.Context) error {
	if !c.consulConfigurer.Enabled {
		logger.InfoC(ctx, "CONSUL_ENABLED is not set. Skip consul integration")
		return nil
	}

	authMode, err := c.resolveConsulAuthMode(ctx)
	if err != nil {
		return utils.LogError(logger, ctx, "error choosing how config-server gets its Consul token: %w", err)
	}

	if authMode == consulauth.AuthModeK8s {
		logger.InfoC(ctx, "Consul auth mode is '%s' and no static token is needed: deleting secret '%s'", c.auth.Mode, c.secretName)
		if err := c.deleteSecret(ctx, c.Namespace, c.secretName); err != nil {
			return utils.LogError(logger, ctx, "error deleting secret %s: %w", c.secretName, err)
		}
		return nil
	}

	if err := c.issueToken(ctx); err != nil {
		return utils.LogError(logger, ctx, "error configure config-server access to consul: %w", err)
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
