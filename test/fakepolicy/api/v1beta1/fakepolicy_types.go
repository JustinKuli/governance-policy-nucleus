// Copyright Contributors to the Open Cluster Management project

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nucleusv1beta1 "open-cluster-management.io/governance-policy-nucleus/api/v1beta1"
)

// FakePolicySpec defines the desired state of FakePolicy
type FakePolicySpec struct {
	nucleusv1beta1.PolicyCoreSpec `json:",inline"`

	// targetConfigMaps defines the ConfigMaps which should be examined by this policy
	TargetConfigMaps nucleusv1beta1.Target `json:"targetConfigMaps,omitempty"`

	// targetUsingReflection defines whether to use reflection to find the ConfigMaps
	TargetUsingReflection bool `json:"targetUsingReflection,omitempty"`
}

//+kubebuilder:validation:Optional

// FakePolicyStatus defines the observed state of FakePolicy
type FakePolicyStatus struct {
	nucleusv1beta1.PolicyCoreStatus `json:",inline"`

	// SelectedNamespaces stores the list of namespaces the policy applies to
	SelectedNamespaces []string `json:"selectedNamespaces"`

	// DynamicSelectedConfigMaps stores the list of configmaps the policy applies to, as determined
	// by `GetMatchesDynamic`
	DynamicSelectedConfigMaps []string `json:"dynamicSelectedConfigMaps"`

	// ClientSelectedConfigMaps stores the list of configmaps the policy applies to, as determined
	// by `GetMatches`
	ClientSelectedConfigMaps []string `json:"clientSelectedConfigMaps"`

	// SelectionComplete stores whether the selection has been completed
	SelectionComplete bool `json:"selectionComplete"`

	// SelectionError stores the error from the selection, if one occurred
	SelectionError string `json:"selectionError"`
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
