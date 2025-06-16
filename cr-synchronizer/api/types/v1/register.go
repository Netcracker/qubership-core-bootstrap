package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	GroupName       = "core.qubership.org"
	GroupVersion    = "v1"
	CdnGroupName    = "cdn.qubership.org"
	CdnGroupVersion = "v1"
)

var (
	SchemeGroupVersion    = schema.GroupVersion{Group: GroupName, Version: GroupVersion}
	CdnSchemeGroupVersion = schema.GroupVersion{Group: CdnGroupName, Version: CdnGroupVersion}
	SchemeBuilder         = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme           = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
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
	scheme.AddKnownTypes(CdnSchemeGroupVersion,
		&CDN{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	metav1.AddToGroupVersion(scheme, CdnSchemeGroupVersion)

	return nil
}
