package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

type Lockserver struct {
	Type    LockserverType `json:"type"`
	Address string         `json:"address"`
	Path    string         `json:"path"`
}

type LockserverType string

const (
	LockserverTypeEtcd LockserverType = "etcd"
)

const LockserverTypeDefault LockserverType = LockserverTypeEtcd

type ResourceSelector struct {
	// The label key that the selector applies to.
	Key string `json:"key"`
	// Represents a key's relationship to a set of values.
	// Valid operators are In, NotIn, Exists, DoesNotExist
	Operator ResourceSelectorOperator `json:"operator"`
	// An array of string values. If the operator is In or NotIn,
	// the values array must be non-empty. If the operator is Exists or DoesNotExist,
	// This array is replaced during a strategic merge patch.
	// +optional
	Values []string `json:"values,omitempty" protobuf:"bytes,3,rep,name=values"`
}

type ResourceSelectorOperator string

const (
	ResourceSelectorOpIn           ResourceSelectorOperator = "In"
	ResourceSelectorOpNotIn        ResourceSelectorOperator = "NotIn"
	ResourceSelectorOpExists       ResourceSelectorOperator = "Exists"
	ResourceSelectorOpDoesNotExist ResourceSelectorOperator = "DoesNotExist"
)

type TabletContainers struct {
	VTTablet *VTContainer `json:"vttablet"`
	MySQL    *VTContainer `json:"mysql"`
}

type VTContainer struct {
	Image string `json:"image"`

	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type KeyRange struct {
	From string `json:"from"`

	To string `json:"to"`
}
