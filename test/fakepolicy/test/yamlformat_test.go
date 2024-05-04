package test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	nucleusv1beta1 "open-cluster-management.io/governance-policy-nucleus/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("FakePolicy resource format verification", func() {
	sampleYAML := fromTestdata("fakepolicy-sample.yaml")
	extraFieldYAML := fromTestdata("extra-field.yaml")
	emptyMatchExpressionsYAML := fromTestdata("empty-match-expressions.yaml")

	sample := sampleFakePolicy()

	emptyInclude := sampleFakePolicy()
	emptyInclude.Spec.NamespaceSelector.Include = []nucleusv1beta1.NonEmptyString{}

	emptyLabelSelector := sampleFakePolicy()
	emptyLabelSelector.Spec.NamespaceSelector.LabelSelector = &metav1.LabelSelector{}

	nilLabelSelector := sampleFakePolicy()
	nilLabelSelector.Spec.NamespaceSelector.LabelSelector = nil

	emptyMatchExpressions := sampleFakePolicy()
	emptyMatchExpressions.Spec.NamespaceSelector.LabelSelector.MatchExpressions = []metav1.LabelSelectorRequirement{}

	emptyNSSelector := sampleFakePolicy()
	emptyNSSelector.Spec.NamespaceSelector = nucleusv1beta1.NamespaceSelector{}

	emptySeverity := sampleFakePolicy()
	emptySeverity.Spec.Severity = ""

	emptyRemAction := sampleFakePolicy()
	emptyRemAction.Spec.RemediationAction = ""

	reqSelector := sampleFakePolicy()
	reqSelector.Spec.NamespaceSelector.LabelSelector.MatchExpressions = []metav1.LabelSelectorRequirement{{
		Key:      "sample",
		Operator: metav1.LabelSelectorOpExists,
	}}

	// input is a clientObject so that either an Unstructured or the "real" type can be provided.
	DescribeTable("Verifying spec stability", func(input client.Object, wantFile string) {
		Expect(cleanlyCreate(input)).To(Succeed())

		nn := getNamespacedName(input)
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

		wantUnstruct := fromTestdata(wantFile)
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
