package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	"strings"
)

const (
	GroupVersion    = "v1"
	CdnGroupVersion = "v1"
)

var CoreApiGroupNames []string
var CdnApiGroupNames []string

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func init() {
	coreApiGroupNames := "core.qubership.org"
	if coreApiGroupNamesEnv, present := os.LookupEnv("K8S_CORE_API_GROUP_NAMES"); present {
		coreApiGroupNames = coreApiGroupNamesEnv
	}
	CoreApiGroupNames = strings.Split(coreApiGroupNames, ",")

	cdnApiGroupNames := "cdn.qubership.org"
	if cdnApiGroupNamesEnv, present := os.LookupEnv("K8S_CDN_API_GROUP_NAMES"); present {
		cdnApiGroupNames = cdnApiGroupNamesEnv
	}
	CdnApiGroupNames = strings.Split(cdnApiGroupNames, ",")
}

func addKnownTypes(scheme *runtime.Scheme) error {
	for _, baseGroupName := range CoreApiGroupNames {
		baseSchemeGroupVersion := schema.GroupVersion{Group: baseGroupName, Version: GroupVersion}
		scheme.AddKnownTypes(baseSchemeGroupVersion,
			&Mesh{},
			&MeshList{},
			&MaaS{},
			&MaaSList{},
			&DBaaS{},
			&DBaaSList{},
			&Composite{},
			&CompositeList{},
			&Security{},
			&SecurityList{},
			&ConfigurationPackage{},
			&ConfigurationPackageList{},
			&SmartplugPlugin{},
			&SmartplugPluginList{},
			&Gateway{},
			&GatewayList{},
		)
		metav1.AddToGroupVersion(scheme, baseSchemeGroupVersion)
	}

	for _, cdnApiGroupName := range CdnApiGroupNames {
		cdnSchemeGroupVersion := schema.GroupVersion{Group: cdnApiGroupName, Version: CdnGroupVersion}
		scheme.AddKnownTypes(cdnSchemeGroupVersion,
			&CDN{},
		)
		metav1.AddToGroupVersion(scheme, cdnSchemeGroupVersion)
	}

	return nil
}
