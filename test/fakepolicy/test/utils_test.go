// Copyright Contributors to the Open Cluster Management project

package test

import (
	"embed"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nucleusv1beta1 "open-cluster-management.io/governance-policy-nucleus/api/v1beta1"
	fakev1beta1 "open-cluster-management.io/governance-policy-nucleus/test/fakepolicy/api/v1beta1"
)

// cleanlyCreate creates the given object, and registers a callback to delete the object which
// Ginkgo will call at the appropriate time. The error from the `Create` call is returned (so it can
// be checked) and the `Delete` callback handles 'NotFound' errors as a success.
func cleanlyCreate(obj client.Object) error {
	createErr := k8sClient.Create(ctx, obj)

	if createErr == nil {
		DeferCleanup(func() {
			GinkgoWriter.Printf("Deleting %v %v/%v\n", obj.GetObjectKind().GroupVersionKind().Kind,
				obj.GetNamespace(), obj.GetName())
			if err := k8sClient.Delete(ctx, obj); err != nil {
				if !errors.IsNotFound(err) {
					// Use Fail in order to provide a custom message with useful information
					Fail(fmt.Sprintf("Expected success or 'NotFound' error, got %v", err), 1)
				}
			}
		})
	}

	return createErr
}

func getNamespacedName(obj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

//go:embed testdata/*
var testfiles embed.FS

// Unmarshals the given YAML file in testdata/ into an unstructured.Unstructured
func fromTestdata(name string) unstructured.Unstructured {
	objYAML, err := testfiles.ReadFile("testdata/" + name)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	m := make(map[string]interface{})
	ExpectWithOffset(1, yaml.UnmarshalStrict(objYAML, &m)).To(Succeed())

	return unstructured.Unstructured{Object: m}
}

func sampleFakePolicy() fakev1beta1.FakePolicy {
	return fakev1beta1.FakePolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy.open-cluster-management.io/v1beta1",
			Kind:       "FakePolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fakepolicy-sample",
			Namespace: "default",
		},
		Spec: fakev1beta1.FakePolicySpec{
			PolicyCoreSpec: nucleusv1beta1.PolicyCoreSpec{
				Severity:          "low",
				RemediationAction: "inform",
				NamespaceSelector: nucleusv1beta1.NamespaceSelector{
					LabelSelector: &metav1.LabelSelector{},
					Include:       []nucleusv1beta1.NonEmptyString{"*"},
					Exclude:       []nucleusv1beta1.NonEmptyString{"kube-*"},
				},
			},
			TargetConfigMaps: nucleusv1beta1.Target{
				LabelSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{{
						Key:      "sample",
						Operator: metav1.LabelSelectorOpExists,
					}},
				},
			},
		},
	}
}
