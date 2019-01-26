package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

// VitessBackupConfigSpec defines the desired state of VitessBackupConfig
type VitessBackupConfigSpec struct {
	Storage *VitessBackupConfigStorage `json:"storage,omitempty"`

	// parent is unexported on purpose.
	// It should only be used during processing and never stored
	parent VitessBackupConfigParents
}

type VitessBackupConfigStorage struct {
	Implementation VitessBackupStorageImplementation `json:"storage,omitempty"`

	GCS *VitessBackupStorageGCS `json:"gcs,omitempty"`

	S3 *VitessBackupStorageS3 `json:"s3,omitempty"`
}

type VitessBackupStorageImplementation string

const (
	VitessBackupStorageImplementionGCS VitessBackupStorageImplementation = "gcs"
	VitessBackupStorageImplementionS3  VitessBackupStorageImplementation = "s3"
)

type VitessBackupStorageGCS struct {
	Bucket string `json:"bucket"`

	Root string `json:"root"`
}

type VitessBackupStorageS3 struct {
	AwsRegion string `json:"awsRegion,omitempty"`

	Bucket string `json:"bucket"`

	Root string `json:"root"`

	ServerSideEncryption bool `json:"serverSideEncryption"`
}

type VitessBackupConfigParents struct {
	Cluster *VitessCluster
}

// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessBackupConfig is the Schema for the VitessBackupConfigs API
// +k8s:openapi-gen=true
type VitessBackupConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VitessBackupConfigSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessBackupConfigList contains a list of VitessBackupConfig
type VitessBackupConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VitessBackupConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VitessBackupConfig{}, &VitessBackupConfigList{})
}
