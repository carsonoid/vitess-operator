package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

// VitessTabletSpec defines the desired state of VitessTablet
type VitessTabletSpec struct {
	TabletID int64 `json:"tabletId"`

	Cell string `json:"cell"`

	Keyrange KeyRange `json:"keyrange"`

	Type TabletType `json:"type"`

	Datastore TabletDatastore `json:"datastore"`

	Containers TabletContainers `json:"containers"`

	VolumeClaim *corev1.PersistentVolumeClaimVolumeSource `json:"volumeclaim, omitempty"`

	Credentials *TabletCredentials `json:"credentials,omitempty"`
}

type TabletType string

const (
	TabletTypeMaster   TabletType = "master"
	TabletTypeReplica  TabletType = "replica"
	TabletTypeReadOnly TabletType = "readonly"
	TabletTypeBackup   TabletType = "backup"
	TabletTypeRestore  TabletType = "restore"
	TabletTypeDrained  TabletType = "drained"
)

const TabletTypeDefault TabletType = TabletTypeReplica

type TabletDatastore struct {
	Type TabletDatastoreType `json:"type"`
}

type TabletDatastoreType string

const (
	TabletDatastoreTypeLocal TabletDatastoreType = "local"
)

const TabletDatastoreTypeDefault TabletDatastoreType = TabletDatastoreTypeLocal

type TabletContainers struct {
	VTTablet VTContainer `json:"vttablet"`
	MySQL    VTContainer `json:"mysql"`
}

type TabletCredentials struct {
	// SecretRef points a Secret resource which contains the credentials
	// +optional
	SecretRef *corev1.SecretReference `json:"secretRef,omitempty" protobuf:"bytes,4,opt,name=secretRef"`
}

// VitessTabletStatus defines the observed state of VitessTablet
type VitessTabletStatus struct {
	State string `json:"state,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessTablet is the Schema for the vitesstablets API
// +k8s:openapi-gen=true
type VitessTablet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VitessTabletSpec   `json:"spec,omitempty"`
	Status VitessTabletStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessTabletList contains a list of VitessTablet
type VitessTabletList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VitessTablet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VitessTablet{}, &VitessTabletList{})
}