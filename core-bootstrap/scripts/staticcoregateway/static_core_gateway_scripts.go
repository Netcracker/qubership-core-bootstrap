package staticcoregateway

import (
	"context"
	"github.com/netcracker/core-bootstrap/v2/utils"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
)

type deleteK8sResourceAction struct {
	resourceName           string
	deleteResourceFunction func(ctx context.Context, namespace string, resourceName string) error
}

var (
	logger                    = logging.GetLogger("static-core-gateway")
	deleteK8sResourcesActions = []deleteK8sResourceAction{
		{"static-core-gateway-pod-monitor", utils.DeleteK8sPodMonitor},
		{"static-core-gateway", utils.DeleteK8sHorizontalPodAutoscalerV2},
		{"static-core-gateway", utils.DeleteK8sHorizontalPodAutoscalerV2Beta2},
		{"static-core-gateway", utils.DeleteK8sDeployment},
		{"static-core-gateway-service", utils.DeleteK8sService},
		{"static-core-gateway.monitoring-config", utils.DeleteK8sConfigMap},
		{"config-server-internal", utils.DeleteK8sService},
		{"control-plane-internal", utils.DeleteK8sService},
		{"core-operator-internal", utils.DeleteK8sService},
		{"dbaas-agent-internal", utils.DeleteK8sService},
		{"identity-provider-internal", utils.DeleteK8sService},
		{"idp-extensions-internal", utils.DeleteK8sService},
		{"key-manager-internal", utils.DeleteK8sService},
		{"maas-agent-internal", utils.DeleteK8sService},
		{"paas-mediation-internal", utils.DeleteK8sService},
		{"site-management-internal", utils.DeleteK8sService},
		{"staas-agent-internal", utils.DeleteK8sService},
		{"tenant-manager-internal", utils.DeleteK8sService},
	}
)

type Configurer struct {
	Namespace string
}

func New() *Configurer {
	return &Configurer{}
}

func (c *Configurer) Configure(accessor func(string) string) error {
	c.Namespace = utils.MustGetEnv(accessor, "NAMESPACE")
	return nil
}

func (c *Configurer) Execute(ctx context.Context) error {
	logger.InfoC(ctx, "*** Starting static_core_gateway_scripts ***")
	logger.InfoC(ctx, "Starting delete all static-core-gateway K8s resources in namespace: %s", c.Namespace)

	for _, deleteK8sResourceAction := range deleteK8sResourcesActions {
		if err := deleteK8sResourceAction.deleteResourceFunction(ctx, c.Namespace, deleteK8sResourceAction.resourceName); err != nil {
			return err
		}
	}

	logger.InfoC(ctx, "Finished delete all static-core-gateway K8s resources in namespace: %s", c.Namespace)
	logger.InfoC(ctx, "### Finished static_core_gateway_scripts ***")
	return nil
}
