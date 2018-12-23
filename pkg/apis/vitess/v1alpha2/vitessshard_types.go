package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

// VitessShardSpec defines the desired state of VitessShard
type VitessShardSpec struct {
	Defaults VitessShardOptions `json:"defaults"`

	KeyRange KeyRange `json:"keyrange"`

	Tablets []VitessTablet `json:"tablets"`

	TabletSelector []ResourceSelector `json:"tabletSelector,omitempty"`
}

// VitessShardStatus defines the observed state of VitessShard
type VitessShardStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessShard is the Schema for the vitessshards API
// +k8s:openapi-gen=true
type VitessShard struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VitessShardSpec   `json:"spec,omitempty"`
	Status VitessShardStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessShardList contains a list of VitessShard
type VitessShardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VitessShard `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VitessShard{}, &VitessShardList{})
}
