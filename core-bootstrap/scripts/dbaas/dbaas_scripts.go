package dbaas

import (
	"context"
	"fmt"
	"github.com/netcracker/core-bootstrap/v2/utils"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"strconv"
	"strings"
	"time"
)

var logger = logging.GetLogger("dbaas")

type Configurer struct {
	Namespace                    string
	ApiDbaasAddress              string
	Username                     string
	password                     string
	GlobalAutobalanceRules       []string
	MicroserviceAutobalanceRules string
}

type DbConnectionProperties struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Role     string `json:"role"`
	Name     string `json:"name"`
	Url      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	TLS      string `json:"tls"`
}

type DatabaseResponse struct {
	Name                 string                 `json:"name"`
	ConnectionProperties DbConnectionProperties `json:"connectionProperties"`
}

func New() *Configurer {
	return &Configurer{}
}

func (c *Configurer) Configure(accessor func(string) string) error {
	c.Namespace = utils.MustGetEnv(accessor, "NAMESPACE")
	c.ApiDbaasAddress = utils.MustGetEnv(accessor, "API_DBAAS_ADDRESS")
	c.Username = utils.MustGetEnv(accessor, "DBAAS_CLUSTER_DBA_CREDENTIALS_USERNAME")
	c.password = utils.MustGetEnv(accessor, "DBAAS_CLUSTER_DBA_CREDENTIALS_PASSWORD")

	raw := accessor("DBAAS_LODB_PER_NAMESPACE_AUTOBALANCE_RULES")
	c.GlobalAutobalanceRules = strings.Split(strings.ReplaceAll(raw, " ", ""), "||")

	c.MicroserviceAutobalanceRules = accessor("DBAAS_ON_MICROSERVICES_PHYSDB_RULE")
	return nil
}

func (c *Configurer) Execute(ctx context.Context) error {
	if len(c.GlobalAutobalanceRules) > 0 && c.GlobalAutobalanceRules[0] != "" {
		if err := c.createDbaasAutoBalanceRulesOnNamespace(ctx); err != nil {
			return fmt.Errorf("error apply dbaas rules: %w", err)
		}
	} else {
		logger.InfoC(ctx, "No DBAAS_LODB_PER_NAMESPACE_AUTOBALANCE_RULES found, skipping CreateDbaasAutoBalanceRulesOnNamespace")
	}

	if c.MicroserviceAutobalanceRules != "" {
		err := c.createDbaasAutoBalanceRulesOnMs(ctx)
		if err != nil {
			return utils.LogError(logger, ctx, "error during CreateDbaasAutoBalanceRulesOnMs: %w", err)
		}
	} else {
		logger.InfoC(ctx, "No DBAAS_ON_MICROSERVICES_PHYSDB_RULE, skipping CreateDbaasAutoBalanceRulesOnMs")
		return nil
	}

	return nil
}

func (c *Configurer) CreateDatabase(ctx context.Context, microserviceName string, secretName string, namingMapper map[string]string) error {
	dbProperties, err := c.getOrCreateDb(ctx, microserviceName)
	if err != nil {
		return fmt.Errorf("error get or create database for `%s': %w", microserviceName, err)
	}

	data := map[string][]byte{
		mapSecretName("dbhostname", namingMapper): []byte(dbProperties.Host),
		mapSecretName("dbport", namingMapper):     []byte(strconv.Itoa(dbProperties.Port)),
		mapSecretName("role", namingMapper):       []byte(dbProperties.Role),
		mapSecretName("dbname", namingMapper):     []byte(dbProperties.Name),
		mapSecretName("url", namingMapper):        []byte(dbProperties.Url),
		mapSecretName("username", namingMapper):   []byte(dbProperties.Username),
		mapSecretName("password", namingMapper):   []byte(dbProperties.Password),
		mapSecretName("tls", namingMapper):        []byte(dbProperties.TLS),
	}

	return utils.CreateSecretWithDbCredsData(ctx, c.Namespace, secretName, data)
}

func mapSecretName(name string, namingMapper map[string]string) string {
	if namingMapper == nil {
		return name
	}

	newName, ok := namingMapper[name]
	if !ok {
		return name
	} else {
		return newName
	}
}

func (c *Configurer) getOrCreateDb(ctx context.Context, microserviceName string) (DbConnectionProperties, error) {
	dbaasCreateDbURL := fmt.Sprintf("%s/api/v3/dbaas/%s/databases", c.ApiDbaasAddress, c.Namespace)
	logger.InfoC(ctx, fmt.Sprintf("Registering %s database in DbaaS, URL: %s", microserviceName, dbaasCreateDbURL))

	classifier := map[string]string{
		"namespace":        c.Namespace,
		"microserviceName": microserviceName,
		"scope":            "service",
	}
	databaseToRegister := map[string]interface{}{
		"classifier":    classifier,
		"originService": microserviceName,
		"type":          "postgresql",
	}

	var dbResponse DatabaseResponse
	maxAttempts := 10
	for attempts := 0; attempts < maxAttempts; attempts++ {
		resp, err := utils.RestyClient.R().
			SetContext(ctx).
			SetBasicAuth(c.Username, c.password).
			SetHeader("Content-Type", "application/json").
			SetBody(databaseToRegister).
			SetResult(&dbResponse).
			Put(dbaasCreateDbURL)

		if err != nil {
			return DbConnectionProperties{}, utils.LogError(logger, ctx, "Error sending request to dbaas: %v", err)
		}

		if resp.StatusCode() == 202 {
			logger.InfoC(ctx, "Got 202 ACCEPTED response, retrying in 3 seconds...")
			time.Sleep(3 * time.Second)
			continue
		}

		if resp.StatusCode() != 200 && resp.StatusCode() != 201 {
			return DbConnectionProperties{}, utils.LogError(logger, ctx, "Wrong dbaas response status: %s, resp: %s", resp.Status(), resp.String())
		}

		if resp.StatusCode() == 200 {
			logger.InfoC(ctx, "Database already exists, skipping creation")
		}

		logger.InfoC(ctx, fmt.Sprintf("Database creation successful: %+v", dbResponse))
		break
	}

	if dbResponse.ConnectionProperties.Host == "" || dbResponse.ConnectionProperties.Username == "" || dbResponse.ConnectionProperties.Password == "" {
		return DbConnectionProperties{}, utils.LogError(logger, ctx, "some database connection properties are missing, cannot prepare database: %v", dbResponse)
	}

	return dbResponse.ConnectionProperties, nil
}

// https://perch.qubership.org/display/CLOUDCORE/How+to+configure+namespace+autobalance+rules
func (c *Configurer) createDbaasAutoBalanceRulesOnNamespace(ctx context.Context) error {
	logger.InfoC(ctx, "*** starting CreateDbaasAutoBalanceRulesOnNamespace")

	for _, entry := range c.GlobalAutobalanceRules {

		ruleParts := strings.Split(entry, "=>")
		if len(ruleParts) != 2 {
			return utils.LogError(logger, ctx, "Invalid rule format: %s, skipping", entry)
		}

		dbType := ruleParts[0]
		phyDbID := ruleParts[1]

		ruleName := fmt.Sprintf("%s-%s", c.Namespace, dbType)
		ruleJSON := fmt.Sprintf(`{
			"type": "%s",
			"rule": {
				"config": {
					"perNamespace": {
						"phydbid": "%s"
					}
				},
				"type": "perNamespace"
			}
		}`, dbType, phyDbID)

		aggregatorRulesURL := fmt.Sprintf("%s/api/v3/dbaas/%s/physical_databases/balancing/rules/%s", c.ApiDbaasAddress, c.Namespace, ruleName)

		logger.InfoC(ctx, "Sending dbaas auto balancing rule %s: %s, url: %s", ruleName, ruleJSON, aggregatorRulesURL)
		resp, err := utils.RestyClient.R().
			SetContext(ctx).
			SetBasicAuth(c.Username, c.password).
			SetHeader("Content-Type", "application/json").
			SetBody(ruleJSON).
			Put(aggregatorRulesURL)

		if err != nil {
			logger.ErrorC(ctx, "Error creating DBaaS per namespace balancing rule: %v", err)
			return err
		}

		if resp.StatusCode() != 200 && resp.StatusCode() != 201 {
			logger.InfoC(ctx, "Error creating DBaaS per namespace balancing rule [HTTP status: %d]", resp.StatusCode())
			logger.InfoC(ctx, "[HTTP body: %s]", resp.String())
			return fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode())
		}

		logger.InfoC(ctx, "DBaaS auto balancing rule '%s' created successfully", ruleName)
	}

	logger.InfoC(ctx, "*** finishing CreateDbaasAutoBalanceRulesOnNamespace")
	return nil
}

// https://perch.qubership.org/display/CLOUDCORE/On+Microservice+physical+DB+balancing+rule
func (c *Configurer) createDbaasAutoBalanceRulesOnMs(ctx context.Context) error {
	logger.InfoC(ctx, "*** starting CreateDbaasAutoBalanceRulesOnMs")

	aggregatorRulesURL := fmt.Sprintf("%s/api/v3/dbaas/%s/physical_databases/rules/onMicroservices", c.ApiDbaasAddress, c.Namespace)
	logger.InfoC(ctx, "Sending dbaas auto balancing rule on ms to url: %s", aggregatorRulesURL)

	resp, err := utils.RestyClient.R().
		SetContext(ctx).
		SetBasicAuth(c.Username, c.password).
		SetHeader("Content-Type", "application/json").
		SetBody(c.MicroserviceAutobalanceRules).
		Put(aggregatorRulesURL)

	if err != nil {
		return utils.LogError(logger, ctx, "Error sending on ms balancing rule: %v", err)
	}

	if resp.StatusCode() != 200 && resp.StatusCode() != 201 {
		logger.InfoC(ctx, "Error creating dbaas on ms balancing rule [HTTP status: %v]", resp.StatusCode())
		logger.InfoC(ctx, "[HTTP body: %s]", resp.String())
		return fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode())
	}

	logger.InfoC(ctx, "DBaaS auto balancing rule created successfully")
	return nil
}
