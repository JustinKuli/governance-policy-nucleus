// Copyright Contributors to the Open Cluster Management project

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nucleusv1beta1 "open-cluster-management.io/governance-policy-nucleus/api/v1beta1"
)

// FakePolicySpec defines the desired state of FakePolicy
type FakePolicySpec struct {
	nucleusv1beta1.PolicyCoreSpec `json:",inline"`

	// TargetConfigMaps defines the ConfigMaps which should be examined by this policy
	TargetConfigMaps nucleusv1beta1.Target `json:"targetConfigMaps,omitempty"`

	// TargetUsingReflection defines whether to use reflection to find the ConfigMaps
	TargetUsingReflection bool `json:"targetUsingReflection,omitempty"`

	// DesiredConfigMapName - if this name is not found, the policy will report a violation
	DesiredConfigMapName string `json:"desiredConfigMapName,omitempty"`

	// EventAnnotation - if provided, this value will be annotated on the compliance
	// events, under the "policy.open-cluster-management.io/test" key
	EventAnnotation string `json:"eventAnnotation,omitempty"`
}

//+kubebuilder:validation:Optional

// FakePolicyStatus defines the observed state of FakePolicy
type FakePolicyStatus struct {
	nucleusv1beta1.PolicyCoreStatus `json:",inline"`

	// SelectionComplete stores whether the selection has been completed
	SelectionComplete bool `json:"selectionComplete"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// FakePolicy is the Schema for the fakepolicies API
type FakePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FakePolicySpec   `json:"spec,omitempty"`
	Status FakePolicyStatus `json:"status,omitempty"`
}

// ensure FakePolicy implements PolicyLike
var _ nucleusv1beta1.PolicyLike = (*FakePolicy)(nil)

func (f FakePolicy) ComplianceState() nucleusv1beta1.ComplianceState {
	return f.Status.ComplianceState
}

func (f FakePolicy) ComplianceMessage() string {
	idx, compCond := f.Status.GetCondition("Compliant")
	if idx == -1 {
		return ""
	}

	return compCond.Message
}

func (f FakePolicy) Parent() metav1.OwnerReference {
	if len(f.OwnerReferences) == 0 {
		return metav1.OwnerReference{}
	}

	return f.OwnerReferences[0]
}

func (f FakePolicy) ParentNamespace() string {
	return f.Namespace
}

//+kubebuilder:object:root=true

// FakePolicyList contains a list of FakePolicy
type FakePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FakePolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FakePolicy{}, &FakePolicyList{})
}
