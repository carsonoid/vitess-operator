package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

// VitessCellSpec defines the desired state of VitessCell
type VitessCellSpec struct {
	Lockserver VitessLockserver `json:"lockserver"`

	Defaults *VitessCellDefaults `json:"defaults"`

	VTGate []VTComponent `json:"vtgate"`

	VTWorker []VTComponent `json:"vtworker"`

	VTCtld []VTComponent `json:"vtctld"`

	Orchestrator []VTComponent `json:"orchestrator"`
}

type VitessCellDefaults struct {
	Replicas *int32 `json:"replicas"`

	Image string `json:"image"`
}

type VTGate struct {
	// Inline common component struct members
	VTComponent `json:",inline"`

	Credentials VTGateCredentials `json:"credentials,omitempty"`

	Cells []string `json:"cells:`

	CellSelector *CellSelector `json:"cellSelector,omitempty"`
}

type VTGateCredentials struct {
	// SecretRef points a Secret resource which contains the credentials
	// +optional
	SecretRef *corev1.SecretReference `json:"secretRef,omitempty" protobuf:"bytes,4,opt,name=secretRef"`
}

type CellSelector struct {
	MatchLabels map[string]string `json:"matchLabels,omitempty"`

	MatchExpressions []ResourceSelector `json:"matchExpressions,omitempty"`
}

type VTComponent struct {
	Replicas int64 `json:"replicas,omitempty"`

	ContainerSpec []*corev1.Container `json:"containerSpec,omitempty"`
}

// VitessCellStatus defines the observed state of VitessCell
type VitessCellStatus struct {
	State string `json:"state,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessCell is the Schema for the vitesscells API
// +k8s:openapi-gen=true
type VitessCell struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VitessCellSpec   `json:"spec,omitempty"`
	Status VitessCellStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessCellList contains a list of VitessCell
type VitessCellList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VitessCell `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VitessCell{}, &VitessCellList{})
}
