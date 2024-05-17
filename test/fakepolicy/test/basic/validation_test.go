// Copyright Contributors to the Open Cluster Management project

package basic

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	. "open-cluster-management.io/governance-policy-nucleus/test/fakepolicy/test/utils"
)

var _ = Describe("FakePolicy CRD Validation", func() {
	DescribeTable("Validating spec inputs",
		func(ctx SpecContext, severity, remediationAction string, include, exclude []string, isValid bool) {
			policy := FromTestdata("fakepolicy-sample.yaml")

			Expect(unstructured.SetNestedField(policy.Object,
				severity, "spec", "severity")).To(Succeed())
			Expect(unstructured.SetNestedField(policy.Object,
				remediationAction, "spec", "remediationAction")).To(Succeed())
			Expect(unstructured.SetNestedStringSlice(policy.Object,
				include, "spec", "namespaceSelector", "include")).To(Succeed())
			Expect(unstructured.SetNestedStringSlice(policy.Object,
				exclude, "spec", "namespaceSelector", "exclude")).To(Succeed())

			if isValid {
				Expect(tk.CleanlyCreate(ctx, &policy)).To(Succeed())
			} else if !errors.IsInvalid(tk.CleanlyCreate(ctx, &policy)) {
				Fail("Expected creating the policy to fail with an 'invalid' error")
			}
		},
		// Test severity options
		Entry("severity=low", "low", "inform", []string{"*"}, []string{"kube-*"}, true),
		Entry("severity=medium", "medium", "inform", []string{"*"}, []string{"kube-*"}, true),
		Entry("severity=high", "high", "inform", []string{"*"}, []string{"kube-*"}, true),
		Entry("severity=critical", "critical", "inform", []string{"*"}, []string{"kube-*"}, true),
		Entry("severity=Low", "Low", "inform", []string{"*"}, []string{"kube-*"}, true),
		Entry("severity=LOW", "LOW", "inform", []string{"*"}, []string{"kube-*"}, false),
		Entry("severity=super", "super", "inform", []string{"*"}, []string{"kube-*"}, false),
		Entry("severity=''", "", "inform", []string{"*"}, []string{"kube-*"}, false),

		// Test remediationAction options
		Entry("remediationAction=inform", "low", "inform", []string{"*"}, []string{"kube-*"}, true),
		Entry("remediationAction=enforce", "low", "enforce", []string{"*"}, []string{"kube-*"}, true),
		Entry("remediationAction=Inform", "low", "Inform", []string{"*"}, []string{"kube-*"}, true),
		Entry("remediationAction=Enforce", "low", "Enforce", []string{"*"}, []string{"kube-*"}, true),
		Entry("remediationAction=INFORM", "low", "INFORM", []string{"*"}, []string{"kube-*"}, false),
		Entry("remediationAction=like-magic", "low", "like-magic", []string{"*"}, []string{"kube-*"}, false),

		// Test namespaceSelector options
		Entry("empty string in namespaceSelector.include", "low", "inform", []string{""}, []string{"kube-*"}, false),
		Entry("empty string in namespaceSelector.exclude", "low", "inform", []string{"*"}, []string{""}, false),
		Entry("empty list in namespaceSelector.include", "low", "inform", []string{}, []string{"kube-*"}, true),
		Entry("empty list in namespaceSelector.exclude", "low", "inform", []string{"*"}, []string{}, true),
	)
})
