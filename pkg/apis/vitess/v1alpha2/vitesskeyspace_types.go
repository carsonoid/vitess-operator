package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

// VitessKeyspaceSpec defines the desired state of VitessKeyspace
type VitessKeyspaceSpec struct {
	Defaults VitessKeyspaceOptions `json:"defaults"`

	Shards []VitessShard `json:"shards"`

	ShardSelector []ResourceSelector `json:"shardSelector,omitempty"`
}

type VitessKeyspaceOptions struct {
	Shards VitessShardOptions `json:"shards"`

	Replicas VitessReplicaOptions `json:"replicas"`

	Batch VitessBatchOptions `json:""batch`

	Containers VitessContainerOptions `json:"containers"`

	Cells []string `json:"cells"`

	CellSelector []ResourceSelector `json:"cellSelector,omitempty"`
}

type VitessShardOptions struct {
	Count int64 `json:"count"`
}

type VitessReplicaOptions struct {
	Count int64 `json:"count"`
}

type VitessBatchOptions struct {
	Count int64 `json:"count"`
}

type VitessContainerOptions struct {
	VTTablet string `json:"vttablet"`
	MySQL    string `json:"mysql"`
}

// VitessKeyspaceStatus defines the observed state of VitessKeyspace
type VitessKeyspaceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessKeyspace is the Schema for the vitesskeyspaces API
// +k8s:openapi-gen=true
type VitessKeyspace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VitessKeyspaceSpec   `json:"spec,omitempty"`
	Status VitessKeyspaceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessKeyspaceList contains a list of VitessKeyspace
type VitessKeyspaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VitessKeyspace `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VitessKeyspace{}, &VitessKeyspaceList{})
}
