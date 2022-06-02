package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

const (
	version = "v1"
)

// PodTerminator constants
const (
	ClusterKind       = "Cluster"
	ClusterName       = "cluster"
	ClusterNamePlural = "clusters"
	MasterKind        = "Master"
	MasterSetKind     = "MasterSet"
	MasterName        = "master"
	MasterNamePlural  = "masters"

	NodePoolKind   = "NodePool"
	NodePoolName   = "nodepool"
	NodePoolPlural = "nodepools"

	MasterSetName       = "masterset"
	MasterSetNamePlural = "mastersets"
	ClusterScope        = "Namespaced"
	GroupName           = "alibabacloud.com"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: version}

var (
	Scheme         = runtime.NewScheme()
	Codecs         = serializer.NewCodecFactory(Scheme)
	ParameterCodec = runtime.NewParameterCodec(Scheme)
)

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return VersionKind(kind).GroupKind()
}

// VersionKind takes an unqualified kind and returns back a Group qualified GroupVersionKind
func VersionKind(kind string) schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind(kind)
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Master{},
		&MasterList{},
		&MasterSet{},
		&MasterSetList{},
		&Cluster{},
		&ClusterList{},
		&NodePool{},
		&NodePoolList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
