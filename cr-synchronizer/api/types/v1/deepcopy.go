package v1

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
)

func deepCopyInto(dst interface{}, src interface{}) {
	if dst == nil {
		panic("dst cannot be nil")
	}
	if src == nil {
		panic("src cannot be nil")
	}
	bytes, err := json.Marshal(src)
	if err != nil {
		panic("Unable to marshal src")
	}
	err = json.Unmarshal(bytes, dst)
	if err != nil {
		panic("Unable to unmarshal into dst")
	}
}

func (in *Mesh) DeepCopy() *Mesh {
	if in == nil {
		return nil
	}
	out := new(Mesh)
	deepCopyInto(out, in)
	return out
}

func (in *MeshList) DeepCopy() *MeshList {
	if in == nil {
		return nil
	}
	out := new(MeshList)
	deepCopyInto(out, in)
	return out
}

func (in *Mesh) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *MeshList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *MeshList) DeepCopyInto(out *MeshList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Mesh, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *Mesh) DeepCopyInto(out *Mesh) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	deepCopyInto(&out.Spec, in.Spec)
}

func (in *MaaS) DeepCopy() *MaaS {
	if in == nil {
		return nil
	}
	out := new(MaaS)
	deepCopyInto(out, in)
	return out
}

func (in *MaaSList) DeepCopy() *MaaSList {
	if in == nil {
		return nil
	}
	out := new(MaaSList)
	deepCopyInto(out, in)
	return out
}

func (in *MaaS) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *MaaSList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *MaaSList) DeepCopyInto(out *MaaSList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MaaS, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *MaaS) DeepCopyInto(out *MaaS) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	deepCopyInto(&out.Spec, in.Spec)
}

func (in *Composite) DeepCopy() *Composite {
	if in == nil {
		return nil
	}
	out := new(Composite)
	deepCopyInto(out, in)
	return out
}

func (in *CompositeList) DeepCopy() *CompositeList {
	if in == nil {
		return nil
	}
	out := new(CompositeList)
	deepCopyInto(out, in)
	return out
}

func (in *Composite) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *CompositeList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *CompositeList) DeepCopyInto(out *CompositeList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Composite, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *Composite) DeepCopyInto(out *Composite) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	deepCopyInto(&out.Spec, in.Spec)
}

func (in *DBaaS) DeepCopy() *DBaaS {
	if in == nil {
		return nil
	}
	out := new(DBaaS)
	deepCopyInto(out, in)
	return out
}

func (in *DBaaSList) DeepCopy() *DBaaSList {
	if in == nil {
		return nil
	}
	out := new(DBaaSList)
	deepCopyInto(out, in)
	return out
}

func (in *DBaaS) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *DBaaSList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *DBaaSList) DeepCopyInto(out *DBaaSList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]DBaaS, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *DBaaS) DeepCopyInto(out *DBaaS) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	deepCopyInto(&out.Spec, in.Spec)
}

func (in *Security) DeepCopy() *Security {
	if in == nil {
		return nil
	}
	out := new(Security)
	deepCopyInto(out, in)
	return out
}

func (in *SecurityList) DeepCopy() *SecurityList {
	if in == nil {
		return nil
	}
	out := new(SecurityList)
	deepCopyInto(out, in)
	return out
}

func (in *Security) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *SecurityList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *SecurityList) DeepCopyInto(out *SecurityList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Security, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *Security) DeepCopyInto(out *Security) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	deepCopyInto(&out.Spec, in.Spec)
}

// ===== CDN =====

func (in *CDN) DeepCopy() *CDN {
	if in == nil {
		return nil
	}
	out := new(CDN)
	deepCopyInto(out, in)
	return out
}

func (in *CDNList) DeepCopy() *CDNList {
	if in == nil {
		return nil
	}
	out := new(CDNList)
	deepCopyInto(out, in)
	return out
}

func (in *CDN) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *CDNList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *CDNList) DeepCopyInto(out *CDNList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CDN, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *CDN) DeepCopyInto(out *CDN) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	deepCopyInto(&out.Spec, in.Spec)
}

// ===== ConfigurationPackage =====

func (in *ConfigurationPackage) DeepCopy() *ConfigurationPackage {
	if in == nil {
		return nil
	}
	out := new(ConfigurationPackage)
	deepCopyInto(out, in)
	return out
}

func (in *ConfigurationPackageList) DeepCopy() *ConfigurationPackageList {
	if in == nil {
		return nil
	}
	out := new(ConfigurationPackageList)
	deepCopyInto(out, in)
	return out
}

func (in *ConfigurationPackage) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *ConfigurationPackageList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *ConfigurationPackageList) DeepCopyInto(out *ConfigurationPackageList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ConfigurationPackage, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *ConfigurationPackage) DeepCopyInto(out *ConfigurationPackage) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	deepCopyInto(&out.Spec, in.Spec)
}

// ===== SmartplugPlugin =====

func (in *SmartplugPlugin) DeepCopy() *SmartplugPlugin {
	if in == nil {
		return nil
	}
	out := new(SmartplugPlugin)
	deepCopyInto(out, in)
	return out
}

func (in *SmartplugPluginList) DeepCopy() *SmartplugPluginList {
	if in == nil {
		return nil
	}
	out := new(SmartplugPluginList)
	deepCopyInto(out, in)
	return out
}

func (in *SmartplugPlugin) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *SmartplugPluginList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *SmartplugPluginList) DeepCopyInto(out *SmartplugPluginList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]SmartplugPlugin, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *SmartplugPlugin) DeepCopyInto(out *SmartplugPlugin) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	deepCopyInto(&out.Spec, in.Spec)
}

// ===== Gateways =====

func (in *Gateway) DeepCopy() *Gateway {
	if in == nil {
		return nil
	}
	out := new(Gateway)
	deepCopyInto(out, in)
	return out
}

func (in *GatewayList) DeepCopy() *GatewayList {
	if in == nil {
		return nil
	}
	out := new(GatewayList)
	deepCopyInto(out, in)
	return out
}

func (in *Gateway) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *GatewayList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *GatewayList) DeepCopyInto(out *GatewayList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Gateway, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *Gateway) DeepCopyInto(out *Gateway) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	deepCopyInto(&out.Spec, in.Spec)
}
