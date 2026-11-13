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
	authSettings     consulauth.AuthSettings
}

func New(consulConfigurer *consul.Configurer) *Configurer {
	return &Configurer{consulConfigurer: consulConfigurer}
}

func (c *Configurer) Configure(accessor func(string) string) error {
	c.Namespace = utils.MustGetEnv(accessor, "NAMESPACE")
	c.secretName = ConfigServerConsulTokenName

	authSettings, err := consulauth.ReadAuthSettings()
	if err != nil {
		return err
	}
	c.authSettings = authSettings
	if c.consulConfigurer.Enabled && authSettings.Mode != consulauth.AuthModeK8s &&
		(c.consulConfigurer.Address == "" || c.consulConfigurer.AdminToken() == "") {
		return fmt.Errorf("consul public URL and admin token are required if CONSUL_ENABLED true and the Consul auth mode is '%s'", authSettings.Mode)
	}
	return nil
}

func (c *Configurer) Execute(ctx context.Context) error {
	if !c.consulConfigurer.Enabled {
		logger.InfoC(ctx, "CONSUL_ENABLED is not set. Skip consul integration")
		return nil
	}

	staticTokenNeeded, err := c.staticTokenNeeded(ctx)
	if err != nil {
		return utils.LogError(logger, ctx, "error choosing how config-server gets its Consul token: %w", err)
	}

	if !staticTokenNeeded {
		logger.InfoC(ctx, "config-server needs no static Consul token: deleting secret '%s'", c.secretName)
		if err := utils.DeleteK8sSecret(ctx, c.Namespace, c.secretName); err != nil {
			return utils.LogError(logger, ctx, "error deleting secret %s: %w", c.secretName, err)
		}
		return nil
	}

	if err := c.configureConsulAccess(ctx); err != nil {
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
