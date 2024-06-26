// Copyright Contributors to the Open Cluster Management project

// Package v1beta1 contains API Schema definitions for the policy v1beta1 API group
// +kubebuilder:object:generate=true
// +groupName=policy.open-cluster-management.io
package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

//nolint:gochecknoglobals // These are traditional
var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "policy.open-cluster-management.io", Version: "v1beta1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
