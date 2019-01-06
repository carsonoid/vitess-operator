package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

// VitessClusterSpec defines the desired state of VitessCluster
type VitessClusterSpec struct {
	Lockserver *VitessLockserverSpec `json:"lockserver,omitempty"`

	LockserverRef *corev1.LocalObjectReference `json:"lockserverRef,omitempty"`

	Cells map[string]*VitessCellSpec `json:"cells,omitempty"`

	CellSelector []ResourceSelector `json:"cellSelector,omitempty"`

	Keyspaces map[string]*VitessKeyspaceSpec `json:"keyspaces,omitempty"`

	KeyspaceSelector []ResourceSelector `json:"keyspaceSelector,omitempty"`
}

// VitessClusterStatus defines the observed state of VitessCluster
type VitessClusterStatus struct {
	State string `json:"state,omitempty"`

	Lockserver *VitessLockserverStatus `json:"lockserver,omitempty"`

	Cells map[string]*VitessCellStatus `json:"cells,omitempty"`

	Keyspaces map[string]*VitessKeyspaceStatus `json:"keyspaces,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessCluster is the Schema for the vitessclusters API
// +k8s:openapi-gen=true
type VitessCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VitessClusterSpec   `json:"spec,omitempty"`
	Status VitessClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessClusterList contains a list of VitessCluster
type VitessClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VitessCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VitessCluster{}, &VitessClusterList{})
}
