// Copyright Contributors to the Open Cluster Management project

package utils

import (
	"embed"

	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"

	nucleusv1beta1 "open-cluster-management.io/governance-policy-nucleus/api/v1beta1"
	fakev1beta1 "open-cluster-management.io/governance-policy-nucleus/test/fakepolicy/api/v1beta1"
)

//go:embed testdata/*
var testfiles embed.FS

// Unmarshals the given YAML file in testdata/ into an unstructured.Unstructured.
func FromTestdata(name string) unstructured.Unstructured {
	objYAML, err := testfiles.ReadFile("testdata/" + name)
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())

	m := make(map[string]interface{})
	gomega.ExpectWithOffset(1, yaml.UnmarshalStrict(objYAML, &m)).To(gomega.Succeed())

	return unstructured.Unstructured{Object: m}
}

func SampleFakePolicy() fakev1beta1.FakePolicy {
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
