package consul

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/netcracker/core-bootstrap/v2/utils"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var logger = logging.GetLogger("consul")

type Policy struct {
	Name        string
	Description string
	Rules       string
}

type TokenSelfResponse struct {
	Policies   []Policy `json:"Policies"`
	AccessorID string   `json:"AccessorID"`
}

type Configurer struct {
	Namespace  string
	Enabled    bool
	Address    string
	adminToken string
}

func New() *Configurer {
	return &Configurer{}
}

func (c *Configurer) Configure(accessor func(string) string) error {
	c.Namespace = utils.MustGetEnv(accessor, "NAMESPACE")
	c.Enabled = utils.GetEnvBoolean(accessor, "CONSUL_ENABLED")
	c.Address = strings.TrimRight(accessor("CONSUL_PUBLIC_URL"), "/")
	c.adminToken = accessor("CONSUL_ADMIN_TOKEN")

	if c.Enabled && (c.Address == "" || c.adminToken == "") {
		return fmt.Errorf("consul public URL and admin token are required if CONSUL_ENABLED true")
	}
	return nil
}

func (c *Configurer) Execute(ctx context.Context) error {
	return nil
}

func GetConsulTokenFromSecret(ctx context.Context, namespace, secretName string) (string, error) {
	secret, err := utils.GetExistingSecret(ctx, namespace, secretName)
	if err != nil {
		return "", utils.LogError(logger, ctx, "error getting token from secret: %v", err)
	}
	if secret == nil {
		logger.InfoC(ctx, "No secret found with token secret named %s", secretName)
		return "", nil
	}

	token, ok := secret.Data["token"]
	if !ok {
		return "", utils.LogError(logger, ctx, "error getting token from secret: %v", secretName)
	}

	return string(token), nil
}

func SaveConsulTokenSecret(ctx context.Context, namespace, token, secretName string) error {
	logger.InfoC(ctx, "Saving secret '%s'...", secretName)
	secretData := map[string][]byte{
		"token": []byte(token),
	}
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
			Labels: map[string]string{
				"app.kubernetes.io/part-of":    "Cloud-Core",
				"app.kubernetes.io/managed-by": "saasDeployer",
			},
		},
		Data: secretData,
		Type: v1.SecretTypeOpaque,
	}
	err := utils.CreateOrUpdateSecret(ctx, namespace, secret)
	if err != nil {
		return utils.LogError(logger, ctx, "error creating or updating secret: %w", err)
	}

	logger.InfoC(ctx, "Secret '%s' created successfully", secretName)
	return nil
}

func SendConsulRequest(ctx context.Context, url, method, token string, payload interface{}, errCodes ...int) (map[string]interface{}, error) {
	if len(errCodes) == 0 {
		errCodes = []int{404}
	}

	resp, err := SendConsulRequestRaw(ctx, url, method, token, payload, errCodes...)
	if err != nil {
		return nil, utils.LogError(logger, ctx, "Error during SendConsulRequestRaw: %w", err)
	}

	if resp == nil {
		return nil, nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

func SendConsulRequestRaw(ctx context.Context, url, method, token string, payload interface{}, errCodes ...int) (*resty.Response, error) {
	resp, err := utils.RestyClient.R().
		SetContext(ctx).
		SetHeader("X-Consul-Token", token).
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		Execute(method, url)

	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	for _, code := range errCodes {
		if resp.StatusCode() == code {
			logger.InfoC(ctx, "url '%s' returned status code %d", url, resp.StatusCode())
			return nil, nil
		}
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode(), resp.String())
	}

	return resp, nil
}

func (c *Configurer) CreateConsulToken(ctx context.Context, policies []Policy, existingToken, secretName string) error {
	logger.InfoC(ctx, "Creating or updating policy and token")

	var payloadPolicies []map[string]string
	for _, policy := range policies {
		payloadPolicies = append(payloadPolicies, map[string]string{"Name": policy.Name})
	}

	tokenPayload := map[string]interface{}{
		"Description": "bootstrap token",
		"Policies":    payloadPolicies,
	}

	var tokenURL string
	if existingToken == "" {
		tokenURL = fmt.Sprintf("%s/v1/acl/token", c.Address)
	} else {
		tokenURL = fmt.Sprintf("%s/v1/acl/token/%s", c.Address, existingToken)
	}

	response, err := SendConsulRequest(ctx, tokenURL, http.MethodPut, c.adminToken, tokenPayload)
	if err != nil {
		return utils.LogError(logger, ctx, "Error during sendConsulRequest: %w", err)
	}

	secretId, ok := response["SecretID"].(string)
	if !ok {
		return utils.LogError(logger, ctx, "error getting SecretID from resp body: %v", response)
	}

	if err := SaveConsulTokenSecret(ctx, c.Namespace, secretId, secretName); err != nil {
		return utils.LogError(logger, ctx, "error store consul token to secret %s: %w", secretName, err)
	}

	return nil
}

func (c *Configurer) LoadPolicyID(ctx context.Context, policyName string) (string, error) {
	logger.InfoC(ctx, "Loading policy ID for policy '%s'", policyName)
	url := fmt.Sprintf("%s/v1/acl/policy/name/%s", c.Address, policyName)

	result, err := SendConsulRequest(ctx, url, http.MethodGet, c.adminToken, nil, 404, 403)
	if err != nil {
		return "", utils.LogError(logger, ctx, "Error getting policy id from url %s: %w", url, err)
	}

	if result == nil {
		logger.InfoC(ctx, "No policy id found for policy '%s'", policyName)
		return "", nil
	}

	policyID, exists := result["ID"].(string)
	if !exists {
		return "", utils.LogError(logger, ctx, "policy ID not found in response")
	}

	return policyID, nil
}

func (c *Configurer) CreateOrUpdatePolicy(ctx context.Context, policyID string, policy Policy) error {
	logger.InfoC(ctx, "Creating or updating Consul policy '%s'", policy.Name)
	policyPayload := map[string]interface{}{
		"Name":        policy.Name,
		"Description": policy.Description,
		"Rules":       policy.Rules,
	}

	var policyUrl string
	if policyID == "" {
		logger.InfoC(ctx, "Policy '%s' does not exist, creating new policy", policy.Name)
		policyUrl = fmt.Sprintf("%s/v1/acl/policy", c.Address)
	} else {
		logger.InfoC(ctx, "Policy '%s' already exists, updating", policy.Name)
		policyUrl = fmt.Sprintf("%s/v1/acl/policy/%s", c.Address, policyID)
	}

	logger.InfoC(ctx, "Policy URL: '%s', policy payload: %+v", policyUrl, policyPayload)

	if _, err := SendConsulRequest(ctx, policyUrl, http.MethodPut, c.adminToken, policyPayload); err != nil {
		return utils.LogError(logger, ctx, "error creating or updating policy: %w", err)
	}

	return nil
}

// last argument is for backward compatibility, check CleanupDuplicateTokens function
func (c *Configurer) CheckAndCreateConsulPoliciesAndToken(ctx context.Context, secretName string, requiredPolicies []Policy, dublicatedPolicy string) error {
	// First check if we have an existing token
	tokenFromSecret, err := GetConsulTokenFromSecret(ctx, c.Namespace, secretName)
	if err != nil {
		return utils.LogError(logger, ctx, "Error getting token from secret: %w", err)
	}

	// For each required policy, ensure it exists and is up to date
	for _, policy := range requiredPolicies {
		policyID, err := c.LoadPolicyID(ctx, policy.Name)
		if err != nil {
			return utils.LogError(logger, ctx, "Error loading policy: %w", err)
		}

		// Create or update the policy
		err = c.CreateOrUpdatePolicy(ctx, policyID, policy)
		if err != nil {
			return utils.LogError(logger, ctx, "Error creating/updating policy: %w", err)
		}
	}

	// If we have an existing token, check if it needs to be updated
	if tokenFromSecret != "" {
		// Check if the token has all required policies
		tokenFromSecret, err = c.CheckRequiredPoliciesOnToken(ctx, tokenFromSecret, requiredPolicies)
		if err != nil {
			return utils.LogError(logger, ctx, "Error checking required policies: %w", err)
		}

		// If token is valid and has all required policies, we're done
		if tokenFromSecret != "" {
			logger.InfoC(ctx, "Token already has required policies, skipping update")
			return nil
		}
	}

	// Clean up any duplicate tokens
	existingToken, err := c.CleanupDuplicateTokens(ctx, dublicatedPolicy)
	if err != nil {
		return utils.LogError(logger, ctx, "Error during CleanupDuplicateTokens: %w", err)
	}

	err = c.CreateConsulToken(ctx, requiredPolicies, existingToken, secretName)
	if err != nil {
		return utils.LogError(logger, ctx, "Error createConsulToken: %w", err)
	}

	return nil
}

func (c *Configurer) CheckRequiredPoliciesOnToken(ctx context.Context, tokenFromSecret string, requiredPolicies []Policy) (string, error) {
	if tokenFromSecret == "" {
		// No token provided; nothing to check.
		return "", nil
	}

	logger.InfoC(ctx, "Checking required policies on token")

	tokenInfo, err := c.GetTokenInfo(ctx, tokenFromSecret)
	if err != nil {
		return "", err
	}

	for _, required := range requiredPolicies {
		found := false
		for _, policy := range tokenInfo.Policies {
			if policy.Name == required.Name {
				found = true
				break
			}
		}
		if !found {
			logger.InfoC(ctx, "Required policy '%s' not found on token. New token will be created.", required.Name)
			return "", nil
		}
	}

	return tokenFromSecret, nil
}

func (c *Configurer) GetTokenInfo(ctx context.Context, token string) (*TokenSelfResponse, error) {
	url := strings.TrimRight(c.Address, "/") + "/v1/acl/token/self"

	respMap, err := SendConsulRequest(ctx, url, "GET", token, nil)
	if err != nil {
		return nil, err
	}
	if respMap == nil {
		return nil, fmt.Errorf("no response received from Consul")
	}

	b, err := json.Marshal(respMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var tokenInfo TokenSelfResponse
	if err := json.Unmarshal(b, &tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token info: %w", err)
	}

	return &tokenInfo, nil
}

// big ugly function for backward compatibility, because config-server coluld have policy with config, but not with log and we need to add it
// futhermore, there was a bug, when several tokens with the same name could exist, that's why we clear them
func (c *Configurer) CleanupDuplicateTokens(ctx context.Context, policyName string) (string, error) {
	tokensURL := strings.TrimRight(c.Address, "/") + "/v1/acl/tokens"

	resp, err := SendConsulRequestRaw(ctx, tokensURL, http.MethodGet, c.adminToken, nil)
	if err != nil {
		return "", fmt.Errorf("failed to list tokens: %w", err)
	}

	var tokens []map[string]interface{}
	{
		if err := json.Unmarshal(resp.Body(), &tokens); err != nil {
			return "", fmt.Errorf("failed to unmarshal tokens list: %w", err)
		}
	}

	var matchingTokens []string
	for _, token := range tokens {
		accessorID, ok := token["AccessorID"].(string)
		if !ok {
			continue
		}
		policies, ok := token["Policies"].([]interface{})
		if !ok {
			continue
		}
		for _, p := range policies {
			pMap, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			if name, ok := pMap["Name"].(string); ok && name == policyName {
				matchingTokens = append(matchingTokens, accessorID)
				break
			}
		}
	}

	tokenCount := len(matchingTokens)
	if tokenCount >= 1 {
		firstToken := matchingTokens[0]
		logger.InfoC(ctx, "Found %d tokens with policy %s. Will keep only the first one (%s) and delete all others.", tokenCount, policyName, firstToken)

		// Delete all tokens except the first one.
		if tokenCount > 1 {
			for _, tokenToDelete := range matchingTokens[1:] {
				deleteURL := fmt.Sprintf("%s/v1/acl/token/%s", strings.TrimRight(c.Address, "/"), tokenToDelete)
				logger.InfoC(ctx, "Deleting token with accessor ID: %s", tokenToDelete)
				if _, err := SendConsulRequest(ctx, deleteURL, http.MethodDelete, c.adminToken, nil); err != nil {
					logger.InfoC(ctx, "Error deleting token %s: %v", tokenToDelete, err)
				}
			}
		}

		logger.InfoC(ctx, "Will update the remaining token with accessor ID: %s", firstToken)
		return firstToken, nil
	}

	logger.InfoC(ctx, "No existing tokens found with policy %s. Will create a new one.", policyName)
	return "", nil
}
