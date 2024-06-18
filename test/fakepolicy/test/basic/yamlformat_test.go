// Copyright Contributors to the Open Cluster Management project

package basic

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nucleusv1beta1 "open-cluster-management.io/governance-policy-nucleus/api/v1beta1"
	"open-cluster-management.io/governance-policy-nucleus/pkg/testutils"
	. "open-cluster-management.io/governance-policy-nucleus/test/fakepolicy/test/utils"
)

var _ = Describe("FakePolicy resource format verification", func() {
	sampleYAML := FromTestdata("fakepolicy-sample.yaml")
	extraFieldYAML := FromTestdata("extra-field.yaml")
	emptyMatchExpressionsYAML := FromTestdata("empty-match-expressions.yaml")

	sample := SampleFakePolicy()

	emptyInclude := SampleFakePolicy()
	emptyInclude.Spec.NamespaceSelector.Include = []nucleusv1beta1.NonEmptyString{}

	emptyLabelSelector := SampleFakePolicy()
	emptyLabelSelector.Spec.NamespaceSelector.LabelSelector = &metav1.LabelSelector{}

	nilLabelSelector := SampleFakePolicy()
	nilLabelSelector.Spec.NamespaceSelector.LabelSelector = nil

	emptyMatchExpressions := SampleFakePolicy()
	emptyMatchExpressions.Spec.NamespaceSelector.LabelSelector.MatchExpressions = []metav1.LabelSelectorRequirement{}

	emptyNSSelector := SampleFakePolicy()
	emptyNSSelector.Spec.NamespaceSelector = nucleusv1beta1.NamespaceSelector{}

	emptySeverity := SampleFakePolicy()
	emptySeverity.Spec.Severity = ""

	emptyRemAction := SampleFakePolicy()
	emptyRemAction.Spec.RemediationAction = ""

	reqSelector := SampleFakePolicy()
	reqSelector.Spec.NamespaceSelector.LabelSelector.MatchExpressions = []metav1.LabelSelectorRequirement{{
		Key:      "sample",
		Operator: metav1.LabelSelectorOpExists,
	}}

	// input is a clientObject so that either an Unstructured or the "real" type can be provided.
	DescribeTable("Verifying spec stability", func(ctx SpecContext, input client.Object, wantFile string) {
		Expect(tk.CleanlyCreate(ctx, input)).To(Succeed())

		nn := testutils.ObjNN(input)
		gotObj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "policy.open-cluster-management.io/v1beta1",
				"kind":       "FakePolicy",
			},
		}

		Expect(k8sClient.Get(ctx, nn, gotObj)).Should(Succeed())

		// Just compare specs; metadata will be different between runs

		gotSpec, err := json.Marshal(gotObj.Object["spec"])
		Expect(err).ToNot(HaveOccurred())

		wantUnstruct := FromTestdata(wantFile)
		wantSpec, err := json.Marshal(wantUnstruct.Object["spec"])
		Expect(err).ToNot(HaveOccurred())

		Expect(string(wantSpec)).To(Equal(string(gotSpec)))
	},
		// The golang instances defined above should match the specified files.
		Entry("The sample YAML policy should be correct",
			sampleYAML.DeepCopy(),
			"fakepolicy-sample.yaml"),
		Entry("The empty matchExpressions should be preserved",
			emptyMatchExpressionsYAML.DeepCopy(),
			"empty-match-expressions.yaml"),
		Entry("An extra field in the spec should be removed",
			extraFieldYAML.DeepCopy(),
			"fakepolicy-sample.yaml"),
		Entry("The sample typed policy should be correct",
			sample.DeepCopy(),
			"fakepolicy-sample.yaml"),
		Entry("An empty Includes list should be removed",
			emptyInclude.DeepCopy(),
			"no-include.yaml"),
		Entry("An empty LabelSelector should have no effect",
			emptyLabelSelector.DeepCopy(),
			"fakepolicy-sample.yaml"),
		Entry("A nil LabelSelector should have no effect",
			nilLabelSelector.DeepCopy(),
			"fakepolicy-sample.yaml"),
		Entry("The emptyMatchExpressions in the typed object should match the YAML",
			emptyMatchExpressions.DeepCopy(),
			"empty-match-expressions.yaml"),
		Entry("An empty NamespaceSelector is not removed",
			emptyNSSelector.DeepCopy(),
			"empty-ns-selector.yaml"),
		Entry("An empty Severity should be removed",
			emptySeverity.DeepCopy(),
			"no-severity.yaml"),
		Entry("An empty RemediationAction should be removed",
			emptyRemAction.DeepCopy(),
			"no-remediation.yaml"),
		Entry("A LabelSelector with Exists doesn't have values",
			reqSelector.DeepCopy(),
			"req-selector.yaml"),
	)
})
