package maas

import (
	"context"
	"errors"
	"fmt"
	"github.com/netcracker/core-bootstrap/v2/utils"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

var logger = logging.GetLogger("maas")

type Configurer struct {
	Namespace string
	Enabled   bool
	Address   string
	Config    string
	Username  string
	password  string
}

func New() *Configurer {
	return &Configurer{}
}

func (c *Configurer) Configure(accessor func(string) string) error {
	c.Namespace = utils.MustGetEnv(accessor, "NAMESPACE")
	c.Enabled = utils.GetEnvBoolean(accessor, "MAAS_ENABLED")
	c.Address = accessor("MAAS_INTERNAL_ADDRESS")
	c.Username = accessor("MAAS_CREDENTIALS_USERNAME")
	c.password = accessor("MAAS_CREDENTIALS_PASSWORD")
	if c.Enabled && c.Address == "" {
		return fmt.Errorf("MAAS_ENABLED set to true, but maas address is not specified via MAAS_INTERNAL_ADDRESS")
	}
	c.Config = accessor("MAAS_CONFIG")
	return nil
}

func (c *Configurer) Execute(ctx context.Context) error {
	if err := c.sendMaaSConfig(ctx); err != nil {
		return utils.LogError(logger, ctx, "error during SendMaaSConfig: %w", err)
	}
	if err := c.maasAgentCreateClient(ctx); err != nil {
		return utils.LogError(logger, ctx, "error during MaasAgentCreateClient: %w", err)
	}
	return nil
}

func (c *Configurer) sendMaaSConfig(ctx context.Context) error {
	logger.InfoC(ctx, "*** starting SendMaaSConfig...")

	if c.Config == "" {
		logger.InfoC(ctx, "no MAAS_CONFIG, skipping sendMaaSConfig")
		return nil
	}
	logger.InfoC(ctx, "config: %s", c.Config)

	aggregatorURL := fmt.Sprintf("%s/api/v2/config", c.Address)
	logger.InfoC(ctx, "Sending maas config to url: %s", aggregatorURL)
	indentedMaasConfig := indent(c.Config, "    ")

	// Build the YAML configuration string.
	configYaml := fmt.Sprintf(`apiVersion: nc.maas.config/v2
kind: config
spec:
  version: v1
  namespace: %s
  shared: |+
%s`, c.Namespace, indentedMaasConfig)

	logger.InfoC(ctx, "aggregated config: %s", configYaml)

	resp, err := utils.RestyClient.R().
		SetContext(ctx).
		SetBasicAuth(c.Username, c.password).
		SetHeader("Content-Type", "application/json").
		SetBody(configYaml).
		Post(aggregatorURL)

	if err != nil {
		return utils.LogError(logger, ctx, "Error sending maas config: %v", err)
	}

	if resp.StatusCode() != 200 && resp.StatusCode() != 201 {
		logger.InfoC(ctx, "Error sending maas config [HTTP status: %d]", resp.StatusCode())
		logger.InfoC(ctx, "[HTTP body: %s]", resp.String())
		return fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode())
	}

	logger.InfoC(ctx, "MaaS config sent successfully")
	return nil
}

func (c *Configurer) maasAgentCreateClient(ctx context.Context) error {
	logger.InfoC(ctx, "*** Starting MaasAgentCreateClient")

	if !c.Enabled {
		logger.InfoC(ctx, "MAAS_ENABLED is not true, creating secret with default credentials.")
		err := c.createMaasAgentSecret(ctx, c.Namespace, "stub-client-not-registered-in-maas", "password")
		if err != nil {
			return utils.LogError(logger, ctx, "Error creating stub secret: %w", err)
		}
		return nil
	}

	existingUsername, err := c.getExistingUsernameFromSecret(ctx, "cluster-maas-agent-credentials-secret")
	if err != nil {
		return utils.LogError(logger, ctx, "Error getting existing username from secret: %w", err)
	}

	if existingUsername != "" && existingUsername != "stub-client-not-registered-in-maas" {
		logger.InfoC(ctx, "secret already exists, skipping MaasAgentCreateClient, existingUsername: %s", existingUsername)
		return nil
	}

	logger.InfoC(ctx, "Secret not found or default credentials detected, creating a new client.")

	newPassword := utils.GeneratePassword(10)
	newUsername := fmt.Sprintf("maas-agent-%s", c.Namespace)

	logger.InfoC(ctx, "Creating new MaaS client: username=%s, maas_address=%s", newUsername, c.Address)

	clientUrl := fmt.Sprintf("%s/api/v1/auth/account/client", c.Address)
	logger.InfoC(ctx, "Sending request to maas '%s' to delete previous maas agent registration", clientUrl)

	deletePayload := map[string]string{
		"username": newUsername,
	}

	deleteResp, err := utils.RestyClient.R().
		SetContext(ctx).
		SetBasicAuth(c.Username, c.password).
		SetHeader("Content-Type", "application/json").
		SetBody(deletePayload).
		Delete(clientUrl)
	if err != nil {
		return utils.LogError(logger, ctx, "Error sending maas agent delete request: %w", err)
	}
	logger.InfoC(ctx, "Received the status from maas: %d", deleteResp.StatusCode())

	logger.InfoC(ctx, "Sending request to maas '%s' to create maas agent registration", clientUrl)
	postPayload := map[string]interface{}{
		"username":  newUsername,
		"password":  newPassword,
		"namespace": c.Namespace,
		"roles":     []string{"agent"},
	}

	postResp, err := utils.RestyClient.R().
		SetContext(ctx).
		SetBasicAuth(c.Username, c.password).
		SetHeader("Content-Type", "application/json").
		SetBody(postPayload).
		Post(clientUrl)
	if err != nil {
		return utils.LogError(logger, ctx, "Error sending maas agent create client request: %w", err)
	}

	logger.InfoC(ctx, "Received the status from maas: %d", postResp.StatusCode())

	if postResp.StatusCode() != 200 && postResp.StatusCode() != 201 {
		return utils.LogError(logger, ctx, "Error during maas agent create client request [HTTP status: %d]", postResp.StatusCode())
	}

	err = c.createMaasAgentSecret(ctx, c.Namespace, newUsername, newPassword)
	if err != nil {
		return utils.LogError(logger, ctx, "failed to create new maas client secret: %w", err)
	}

	logger.InfoC(ctx, "### Finished maas_agent_create_client")
	return nil
}

func (c *Configurer) createMaasAgentSecret(ctx context.Context, namespace, username, password string) error {
	secretName := "cluster-maas-agent-credentials-secret"

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/part-of":    "Cloud-Core",
				"app.kubernetes.io/managed-by": "saasDeployer",
			},
		},
		Data: map[string][]byte{
			"username": []byte(username),
			"password": []byte(password),
		},
		Type: "Opaque",
	}

	err := utils.CreateOrUpdateSecret(ctx, c.Namespace, secret)
	if err != nil {
		return utils.LogError(logger, ctx, "Error creating or updating secret: %w", err)
	}

	logger.InfoC(ctx, "Secret %s created/updated successfully", secretName)
	return nil
}

func (c *Configurer) getExistingUsernameFromSecret(ctx context.Context, secretName string) (string, error) {
	secret, err := utils.GetExistingSecret(ctx, c.Namespace, secretName)
	if err != nil {
		return "", err
	}

	if secret == nil {
		return "", nil
	}

	username := secret.Data["username"]
	if username == nil {
		return "", errors.New("username not found in secret")
	}

	return string(username), nil
}

func indent(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
