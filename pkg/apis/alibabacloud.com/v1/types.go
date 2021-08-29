package v1

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Master struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Config expected cluster specification
	// +optional
	Spec MasterSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// Status cluster current status.
	// +optional
	Status MasterStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MasterList is a top-level list type. The client methods for lists are automatically created.
// You are not supposed to create a separated client for this one.
type MasterList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Master `json:"items"`
}

type MasterStatus struct {
	Peer       []Host   `json:"peer,omitempty" protobuf:"bytes,1,opt,name=peer"`
	BootCFG    *Cluster `json:"bootcfg,omitempty" protobuf:"bytes,2,opt,name=bootcfg"`
	InstanceId string   `json:"instanceId,omitempty" protobuf:"bytes,3,opt,name=instanceId"`
}

type MasterSpec struct {
	IP string `json:"ip,omitempty" protobuf:"bytes,1,opt,name=ip"`
	// ID provider id of type: region.instanceid
	ID   string `json:"id,omitempty" protobuf:"bytes,2,opt,name=id"`
	Role string `json:"role,omitempty" protobuf:"bytes,3,opt,name=role"`
}

func (m *Master) String() string {
	return fmt.Sprintf("master://%s/%s/%s/%s", m.Name, m.Spec.ID, m.Spec.IP, m.Status.InstanceId)
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MasterSetList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []MasterSet `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MasterSet struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Config expected cluster specification
	// +optional
	Spec MasterSetSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// Status cluster current status.
	// +optional
	Status MasterSetStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type MasterSetSpec struct {
	// Relicas is the replica num of master
	Replicas int `json:"replicas,omitempty" protobuf:"bytes,1,opt,name=replicas"`
}

type MasterSetStatus struct {
	InstanceIDS []string `json:"instanceIDS,omitempty" protobuf:"bytes,1,opt,name=instanceIDS"`
	BootCFG     *Cluster `json:"bootCFG,omitempty" protobuf:"bytes,2,opt,name=bootCFG"`
}

type KeyCert struct {
	Key  []byte `json:"key,omitempty" protobuf:"bytes,1,opt,name=key"`
	Cert []byte `json:"cert,omitempty" protobuf:"bytes,2,opt,name=cert"`
}

type Host struct {
	ID string `json:"id,omitempty" protobuf:"bytes,2,opt,name=id"`
	IP string `json:"ip,omitempty" protobuf:"bytes,1,opt,name=ip"`
}

const (
	NODE_ROLE_MASTER = "Master"
	NODE_ROLE_ETCD   = "ETCD"
	NODE_ROLE_WORKER = "Worker"
	NODE_ROLE_HYBRID = "Hybrid"
)

const (
	KUBERNETES_CLUSTER = "kubernetes-cluster"
)

type ClusterId struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              ClusterIdSpec
}

type ClusterIdSpec struct {
	ResourceId string      `json:"resourceId,omitempty" protobuf:"bytes,1,opt,name=resourceId"`
	ExtraRIDs  []string    `json:"extraRIDs,omitempty" protobuf:"bytes,2,opt,name=extraRIDs"`
	CreatedAt  string      `json:"createdAt,omitempty" protobuf:"bytes,3,opt,name=createdAt"`
	UpdatedAt  string      `json:"updatedAt,omitempty" protobuf:"bytes,4,opt,name=updatedAt"`
	Options    *OocOptions `json:"options,omitempty" protobuf:"bytes,5,opt,name=options"`
	Cluster    ClusterSpec `json:"cluster,omitempty" protobuf:"bytes,6,opt,name=cluster"`
}

type ContextCFG struct {
	Kind string `json:"kind,omitempty"`
	// +k8s:conversion-gen=false
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`

	// CurrentContext is the name of the context that you would like to use by default
	CurrentContext string `json:"current-context"`

	// Contexts is a map of referencable names to context configs
	Contexts  []ContextItem  `json:"contexts"`
	Providers []ProviderItem `json:"providers"`
}

type ContextItem struct {
	Name    string   `json:"name"`
	Context *Context `json:"context"`
}

type ProviderItem struct {
	Name     string    `json:"name"`
	Provider *Provider `json:"provider"`
}

type Context struct {
	ProviderKey string `json:"provider-key"`
}

func (in *ContextCFG) CurrentPrvdCFG() *Provider {
	for _, v := range in.Contexts {
		if v.Name == in.CurrentContext {
			for _, p := range in.Providers {
				if p.Name == v.Context.ProviderKey {
					return p.Provider
				}
			}
		}
	}
	// no provider found, should we panic?
	klog.Errorf("no current provider named [%s] found", in.CurrentContext)
	return nil
}

type CommandLineArgs struct {
	WriteTo      string
	OutPutFormat string
}

type OocOptions struct {
	// Endpoint coordinator bootstrap server endpoint
	Endpoint string
	// Role role of nodes.
	Role   string
	Token  string
	Config string

	// BootType 'local' 'coordinator' ''
	BootType string
	// Resource type
	Resource string
	// TargetCount scale target nodes count
	TargetCount int
	ClusterName string

	// Default is an important data structure which contains Context config
	Default *ContextCFG

	// Addons install addons
	// `*` means all.  comma separated
	Addons string

	//Provider          string
	ExpectedMasterCnt int
	// RecoverMode
	RecoverMode string

	//OperatorCFG Config
	OperatorCFG OperatorFlag
}

type OperatorFlag struct {
	MetricsAddr  string
	EnableLeader bool
	BindAddr     string
	Token        string
	MetaConfig   string
	InitialCount int
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Cluster struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Config expected cluster specification
	// +optional
	Spec ClusterSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// Status cluster current status.
	// +optional
	Status ClusterStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type ClusterStatus struct {
	Peers []Host `json:"peers,omitempty" protobuf:"bytes,1,opt,name=peers"`
}

type ClusterSpec struct {
	Bind      BindInfra `json:"iaas,omitempty" protobuf:"bytes,1,opt,name=iaas"`
	ClusterID string    `json:"clusterid,omitempty" protobuf:"bytes,2,opt,name=clusterid"`
	Namespace string    `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
	CloudType string    `json:"cloudType,omitempty" protobuf:"bytes,4,opt,name=cloudType"`

	Network    Network          `json:"network,omitempty" protobuf:"bytes,5,opt,name=network"`
	Etcd       Etcd             `json:"etcd,omitempty" protobuf:"bytes,6,opt,name=etcd"`
	Runtime    ContainerRuntime `json:"runtime,omitempty" protobuf:"bytes,7,opt,name=runtime"`
	Kubernetes Kubernetes       `json:"kubernetes,omitempty" protobuf:"bytes,8,opt,name=kubernetes"`

	AddonInitialized bool     `json:"addonInitialized" protobuf:"bytes,13,opt,name=addonInitialized"`
	Sans             []string `json:"sans,omitempty" protobuf:"bytes,9,opt,name=sans"`
	Token            string   `json:"token,omitempty" protobuf:"bytes,10,opt,name=token"`
	Registry         string   `json:"registry,omitempty" protobuf:"bytes,11,opt,name=registry"`
	Endpoint         Endpoint `json:"endpoint,omitempty" protobuf:"bytes,12,opt,name=endpoint"`
}

type Kubernetes struct {
	Unit
	KubeadmToken string   `json:"kubeadmToken,omitempty" protobuf:"bytes,2,opt,name=kubeadmToken"`
	RootCA       *KeyCert `json:"rootCA,omitempty" protobuf:"bytes,3,opt,name=rootCA"`
	FrontProxyCA *KeyCert `json:"frontProxyCA,omitempty" protobuf:"bytes,4,opt,name=frontProxyCA"`
	SvcAccountCA *KeyCert `json:"serviceAccountCA,omitempty" protobuf:"bytes,5,opt,name=serviceAccountCA"`
	ControlRoot  *KeyCert `json:"controlRoot,omitempty" protobuf:"bytes,6,opt,name=controlRoot"`
}

type Etcd struct {
	Unit
	// Endpoints is etcd peer endpoints. comma separated
	// must be specified when init role is Master|Hybrid
	Endpoints string   `json:"endpoints,omitempty" protobuf:"bytes,1,opt,name=endpoints"`
	InitToken string   `json:"initToken,omitempty" protobuf:"bytes,2,opt,name=initToken"`
	PeerCA    *KeyCert `json:"peerCA,omitempty" protobuf:"bytes,3,opt,name=peerCA"`
	ServerCA  *KeyCert `json:"serverCA,omitempty" protobuf:"bytes,4,opt,name=serverCA"`
}

type ContainerRuntime struct{ Unit }
type Unit struct {
	Name    string            `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	Version string            `json:"version,omitempty" protobuf:"bytes,2,opt,name=version"`
	Paras   map[string]string `json:"paras,omitempty" protobuf:"bytes,3,opt,name=paras"`
}

type Endpoint struct {
	Intranet string `json:"intranet,omitempty" protobuf:"bytes,1,opt,name=intranet"`
	Internet string `json:"internet,omitempty" protobuf:"bytes,2,opt,name=internet"`
}

type Network struct {
	Mode    string `json:"mode,omitempty" protobuf:"bytes,1,opt,name=mode"`
	PodCIDR string `json:"podcidr,omitempty" protobuf:"bytes,2,opt,name=podcidr"`
	SVCCIDR string `json:"svccidr,omitempty" protobuf:"bytes,3,opt,name=svccidr"`
	Domain  string `json:"domain,omitempty" protobuf:"bytes,4,opt,name=domain"`
	NetMask string `json:"netMask,omitempty" protobuf:"bytes,5,opt,name=netMask"`
}

type BindInfra struct {
	Image       string    `json:"image,omitempty" protobuf:"bytes,1,opt,name=image"`
	Disk        Disk      `json:"disk,omitempty" protobuf:"bytes,2,opt,name=disk"`
	Secret      Secret    `json:"secret,omitempty" protobuf:"bytes,3,opt,name=secret"`
	Kernel      Kernel    `json:"kernel,omitempty" protobuf:"bytes,4,opt,name=kernel"`
	Region      string    `json:"region,omitempty" protobuf:"bytes,5,opt,name=region"`
	ZoneId      string    `json:"zoneid,omitempty" protobuf:"bytes,6,opt,name=zoneid"`
	Instance    string    `json:"instance,omitempty" protobuf:"bytes,7,opt,name=instance"`
	WorkerCount int       `json:"workerCount,omitempty" protobuf:"bytes,8,opt,name=workerCount"`
	Provider    *Provider `json:"provider,omitempty" protobuf:"bytes,9,opt,name=provider"`
	ResourceId  string    `json:"resourceId,omitempty" protobuf:"bytes,10,opt,name=resourceId"`
}

type Provider struct {
	//Id   string `json:"id" protobuf:"bytes,2,opt,name=id"`
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// ProviderSource is an iaas provider
	Value json.RawMessage `json:"value"`
}

func (in *Provider) Decode(i interface{}) error { return json.Unmarshal(in.Value, i) }

func ToRawMessage(i interface{}) (json.RawMessage, error) {
	data, err := json.Marshal(i)
	if err != nil {
		return nil, errors.Wrap(err, "marshal to raw message")
	}
	raw := json.RawMessage{}
	err = json.Unmarshal(data, &raw)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal to raw message")
	}
	return raw, nil
}

type Disk struct {
	Size string `json:"size,omitempty" protobuf:"bytes,1,opt,name=size"`
	Type string `json:"type,omitempty" protobuf:"bytes,2,opt,name=type"`
}

type Secret struct {
	Type  string `json:"type,omitempty" protobuf:"bytes,1,opt,name=type"`
	Value Value  `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
}
type Value struct {
	Name     string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	Password string `json:"password,omitempty" protobuf:"bytes,1,opt,name=password"`
}

type Kernel struct {
	Sysctl []string `json:"sysctl,omitempty" protobuf:"bytes,1,opt,name=sysctl"`
}

type Immutable struct {
	CAs CA `json:"cas,omitempty" protobuf:"bytes,1,opt,name=cas"`
}

type CA struct {
	Root      KeyCert `json:"root,omitempty" protobuf:"bytes,1,opt,name=root"`
	FrontRoot KeyCert `json:"frontRoot,omitempty" protobuf:"bytes,2,opt,name=frontRoot"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Cluster `json:"items"`
}

func NewRecoverCluster(
	id, region string,
	prvdCfg *Provider,
) *Cluster {
	return &Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       ClusterKind,
			APIVersion: SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubernetes-cluster",
		},
		Spec: ClusterSpec{
			Bind: BindInfra{
				Region:   region,
				Provider: prvdCfg,
			},
			ClusterID: id,
		},
	}
}

func NewDefaultCluster(
	name string,
	spec ClusterSpec,
) *Cluster {
	return &Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "alibabacloud.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: spec,
	}
}

// NodePoolSpec defines the desired state of NodePool
type NodePoolSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	NodePoolID string `json:"id,omitempty" protobuf:"bytes,1,opt,name=id"`
	AutoHeal   bool   `json:"autoHeal,omitempty" protobuf:"bytes,2,opt,name=autoHeal"`
	Infra      Infra  `json:"infra,omitempty" protobuf:"bytes,3,opt,name=infra"`
}

type Infra struct {
	// DesiredCapacity
	DesiredCapacity int `json:"desiredCapacity,omitempty" protobuf:"bytes,1,opt,name=desiredCapacity"`

	ImageId string            `json:"imageId,omitempty" protobuf:"bytes,2,opt,name=imageId"`
	CPU     int               `json:"cpu,omitempty" protobuf:"bytes,3,opt,name=cpu"`
	Mem     int               `json:"memory,omitempty" protobuf:"bytes,4,opt,name=memory"`
	Tags    map[string]string `json:"tags,omitempty" protobuf:"bytes,5,opt,name=tags"`

	// Generated ref of generated infra ids configmap
	// for provider
	//Generated string

	Bind *BindID `json:"bind,omitempty" protobuf:"bytes,6,opt,name=bind"`
}

// BindID is the infrastructure ids loaded(created) from under BindInfra layer
type BindID struct {
	VswitchIDS      []string `json:"vswitchIDs,omitempty" protobuf:"bytes,1,opt,name=vswitchIDs"`
	ScalingGroupId  string   `json:"scalingGroupId,omitempty" protobuf:"bytes,2,opt,name=scalingGroupId"`
	ConfigurationId string   `json:"configurationId,omitempty" protobuf:"bytes,3,opt,name=configurationId"`
}

// NodePoolStatus defines the observed state of NodePool
type NodePoolStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodePool is the Schema for the nodepools API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodepools,scope=Namespaced
type NodePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodePoolSpec   `json:"spec,omitempty"`
	Status NodePoolStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodePoolList contains a list of NodePool
type NodePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodePool `json:"items"`
}

// RollingSpec defines the desired state of Rolling
type RollingSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	ActiveDeadlineSeconds int   `json:"activeDeadlineSeconds"`
	RestartLimit          int32 `json:"restartLimit,omitempty"`
	SlowStart             bool  `json:"slowStart,omitempty"`
	MaxParallel           int32 `json:"maxParallel,omitempty"`
	MaxUnavailable        int32 `json:"maxUnavailable,omitempty"`
	Paused                bool  `json:"paused,omitempty"`
	//SlowStart             bool                    `json:"slowStart,omitempty"`
	FailurePolicy FailurePolicy `json:"failurePolicy,omitempty"`

	NodeSelector NodesSet       `json:"nodeSelector,omitempty"`
	Type         string         `json:"type,omitempty"`
	TaskSpec     TaskSpec       `json:"taskSpec,omitempty"`
	PodSpec      v1.PodTemplate `json:"podSpec,omitempty"`
}

type RollingTaskSpec struct {
	ConfigTpl ConfigTpl `json:"configTpl,omitempty" protobuf:"bytes,1,opt,name=configTpl"`

	// NodeName
	// node which the task belongs to.
	NodeName string `json:"nodeName,omitempty" protobuf:"bytes,2,opt,name=nodeName"`

	UserData string `json:"userData,omitempty" protobuf:"bytes,3,opt,name=userData"`
}

type NodesSet struct {
	Labels        map[string]string `json:"labels,omitempty"`
	AutoScalingId string            `json:"autoScaling,omitempty"`
	All           bool              `json:"all,omitempty"`
}

// RollingStatus defines the observed state of Rolling
type RollingStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	Phase string `json:"phase,omitempty"`

	// Represents time when the job was acknowledged by the job controller.
	// It is not guaranteed to be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Represents time when the job was completed. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// The number of total running pods.
	Total int32 `json:"total,omitempty"`

	// The number of actively running pods.
	Active int32 `json:"active,omitempty"`

	// The number of pods which reached phase Succeeded.
	Succeeded int32 `json:"succeeded,omitempty"`

	// The number of pods which reached phase Failed.
	Failed int32 `json:"failed,omitempty"`

	// The number of current max parallel.
	CurrentMaxParallel int32 `json:"currentMaxParallel,omitempty"`
}

type FailurePolicy string

const (
	FailurePolicyContinue FailurePolicy = "Continue"
	FailurePolicyFailed   FailurePolicy = "Failed"
	FailurePolicyPause    FailurePolicy = "Pause"
)

const (
	PhaseCompleted   string = "Completed"
	PhasePaused      string = "Paused"
	PhaseFailed      string = "Failed"
	PhaseInitialized        = "Initialized"
	PhaseReconciling        = "Reconciling"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Rolling is the Schema for the rollings API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=rollings,scope=Namespaced
type Rolling struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RollingSpec   `json:"spec,omitempty"`
	Status RollingStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RollingList contains a list of Rolling
type RollingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Rolling `json:"items"`
}

const (
	RollingHashLabel  = "alibabacloud.com/rolling.hash"
	NodePoolHashLabel = "alibabacloud.com/nodepool.hash"
	NodePoolIDLabel   = "alibabacloud.com/nodepool-id"
)

const (
	TaskTypeUpgrade    = "Upgrade"
	TaskTypeAutoRepair = "AutoHeal"
	TaskTypeCommand    = "Command"
	TaskTypePod        = "Pod"
)

// TaskSpec defines the desired state of Task
type TaskSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// ConfigTpl
	// config template which used to make upgrade decision
	// and compute hash to see whether an upgrade is needed.
	ConfigTpl ConfigTpl `json:"configTpl,omitempty" protobuf:"bytes,1,opt,name=configTpl"`

	// TaskType
	// Upgrade| AutoHeal| Command| Pod
	TaskType string `json:"taskType,omitempty" protobuf:"bytes,4,opt,name=taskType"`

	// NodeName
	// node which the task belongs to.
	NodeName string `json:"nodeName,omitempty" protobuf:"bytes,2,opt,name=nodeName"`

	UserData string `json:"userData,omitempty" protobuf:"bytes,3,opt,name=userData"`
}

type ConfigTpl struct {
	ImageId    string  `json:"imageid,omitempty" protobuf:"bytes,1,opt,name=imageid"`
	Runtime    Runtime `json:"runtime,omitempty" protobuf:"bytes,2,opt,name=runtime"`
	Kubernetes Unit    `json:"kubernetes,omitempty" protobuf:"bytes,3,opt,name=kubernetes"`
}

type Runtime struct {
	Name    string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	Version string `json:"version,omitempty" protobuf:"bytes,2,opt,name=version"`
}

// TaskStatus defines the observed state of Task
type TaskStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// Phase Initialized|Reconciling|Completed|Failed
	Phase string `json:"phase,omitempty"`

	// Reason
	// usually failed Reason, might be success info
	Reason string `json:"reason,omitempty"`

	// Hash
	// Last success hash which computed from hash(task.ConfigTpl)
	// Write back hash when reconcile finished to avoid repeated reconcile.
	Hash string `json:"hash,omitempty"`

	// Log error log, for debug only
	Log []byte `json:"log,omitempty"`

	// Progress
	// update progress
	Progress []Progress `json:"progress,omitempty"`
}

type Progress struct {
	Step        string `json:"step,omitempty"`
	Description string `json:"description,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Task is the Schema for the tasks API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=tasks,scope=Namespaced
type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskSpec   `json:"spec,omitempty"`
	Status TaskStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TaskList contains a list of Task
type TaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Task `json:"items"`
}
