package controlplane

import (
	"context"
	"github.com/netcracker/core-bootstrap/v2/utils"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
)

const CpDbCredentialsSecret = "control-plane-db-credentials"

var logger = logging.GetLogger("config-server")

type ControlPlaneConfigurer struct {
	Namespace             string
	databaseCreate        func(context.Context, string, string, map[string]string) error
	cpDbCredentialsSecret string
}

func New(databaseCreate func(context.Context, string, string, map[string]string) error) *ControlPlaneConfigurer {
	return &ControlPlaneConfigurer{databaseCreate: databaseCreate}
}

func (c *ControlPlaneConfigurer) Configure(accessor func(string) string) error {
	c.Namespace = utils.MustGetEnv(accessor, "NAMESPACE")
	c.cpDbCredentialsSecret = accessor("DB_CREDENTIALS_SECRET")
	if c.cpDbCredentialsSecret == "" {
		c.cpDbCredentialsSecret = CpDbCredentialsSecret
	}

	return nil
}

func (c *ControlPlaneConfigurer) Execute(ctx context.Context) error {
	logger.InfoC(ctx, "*** Starting control_plane_prepare_db ***")

	namingMapper := make(map[string]string)
	namingMapper["dbhostname"] = "host"
	namingMapper["dbport"] = "port"
	namingMapper["dbname"] = "database"

	err := c.databaseCreate(ctx, "control-plane", c.cpDbCredentialsSecret, namingMapper)
	if err != nil {
		return utils.LogError(logger, ctx, "Error getting db properties for cp: %v", err)
	}

	logger.InfoC(ctx, "### Finished control_plane_prepare_db ***")
	return nil
}
