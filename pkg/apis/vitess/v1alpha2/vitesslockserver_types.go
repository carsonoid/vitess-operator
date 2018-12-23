package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

// VitessLockserverSpec defines the desired state of VitessLockserver
type VitessLockserverSpec struct {
	Provision bool `json:provision,omitempty`

	Etcd3 *Etcd3Lockserver `json:"etcd3,omitempty"`
}

type Etcd3Lockserver struct {
	Address string `json:"address"`
	Path    string `json:"path"`
}

// VitessLockserverStatus defines the observed state of VitessLockserver
type VitessLockserverStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessLockserver is the Schema for the vitesslockservers API
// +k8s:openapi-gen=true
type VitessLockserver struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VitessLockserverSpec   `json:"spec,omitempty"`
	Status VitessLockserverStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessLockserverList contains a list of VitessLockserver
type VitessLockserverList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VitessLockserver `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VitessLockserver{}, &VitessLockserverList{})
}
