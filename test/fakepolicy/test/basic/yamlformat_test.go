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
	sampleYAML := FromTestdata("policy_v1beta1_fakepolicy.yaml")
	emptyMatchExpressionsYAML := FromTestdata("empty-match-expressions.yaml")
	policycoreSample := FromTestdata("policy_v1beta1_policycore.yaml")

	extraFieldYAML := FromTestdata("extra-field.yaml")

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

		if nn.Name == "policycore-sample" {
			gotObj.Object["kind"] = "PolicyCore"
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
		// These instances should be "stable" - getting them from the cluster after applying them
		// should return the same information (modulo some metadata, of course)
		Entry("The sample fakepolicy YAML should be stable",
			sampleYAML.DeepCopy(),
			"policy_v1beta1_fakepolicy.yaml"),
		Entry("The empty matchExpressions should be preserved",
			emptyMatchExpressionsYAML.DeepCopy(),
			"empty-match-expressions.yaml"),
		Entry("The sample policycore policy should be stable",
			policycoreSample.DeepCopy(),
			"policy_v1beta1_policycore.yaml"),

		// The golang instances defined above should match the specified files.
		Entry("An extra field in the spec should be removed",
			extraFieldYAML.DeepCopy(),
			"policy_v1beta1_fakepolicy.yaml"),
		Entry("The sample typed policy should be correct",
			sample.DeepCopy(),
			"policy_v1beta1_fakepolicy.yaml"),
		Entry("An empty Includes list should be removed",
			emptyInclude.DeepCopy(),
			"no-include.yaml"),
		Entry("An empty LabelSelector should have no effect",
			emptyLabelSelector.DeepCopy(),
			"policy_v1beta1_fakepolicy.yaml"),
		Entry("A nil LabelSelector should have no effect",
			nilLabelSelector.DeepCopy(),
			"policy_v1beta1_fakepolicy.yaml"),
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
