// Copyright Contributors to the Open Cluster Management project

package compliance

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"open-cluster-management.io/governance-policy-nucleus/pkg/testutils"
	fakev1beta1 "open-cluster-management.io/governance-policy-nucleus/test/fakepolicy/api/v1beta1"
	. "open-cluster-management.io/governance-policy-nucleus/test/fakepolicy/test/utils"
)

var _ = Describe("Compliance Events with a Mutator", Ordered, func() {
	const testNS string = "mutator-comp-test"
	const annoKey string = "policy.open-cluster-management.io/test"

	var (
		parent *corev1.ConfigMap
		policy *fakev1beta1.FakePolicy
	)

	BeforeAll(func(ctx SpecContext) {
		ns := &corev1.Namespace{
			// TypeMeta is not required here; the Client can infer it
			ObjectMeta: metav1.ObjectMeta{
				Name: testNS,
			},
		}
		Expect(tk.CleanlyCreate(ctx, ns)).To(Succeed())

		parent = &corev1.ConfigMap{
			// TypeMeta is useful here for the OwnerReference
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "parent",
				Namespace: testNS,
			},
		}
		Expect(tk.CleanlyCreate(ctx, parent)).To(Succeed())

		sample := SampleFakePolicy()
		policy = &sample
		policy.Name += "-mutator-comp-test"
		policy.Namespace = testNS
		policy.Spec.DesiredConfigMapName = "hello-world"
		policy.OwnerReferences = []metav1.OwnerReference{{
			APIVersion: parent.APIVersion,
			Kind:       parent.Kind,
			Name:       parent.Name,
			UID:        parent.UID,
		}}
		Expect(tk.CleanlyCreate(ctx, policy)).To(Succeed())
	})

	It("Should start NonCompliant", func(ctx SpecContext) {
		tk.EC(func(g Gomega) string {
			g.Expect(tk.Get(ctx, testutils.ObjNN(policy), policy)).To(Succeed())

			return string(policy.Status.ComplianceState)
		}, Equal("NonCompliant"))
	})

	It("Should emit one NonCompliant event without the annotation", func(ctx SpecContext) {
		tk.EC(func(g Gomega) []string {
			evs, err := tk.GetComplianceEvents(ctx, testNS, parent.UID, policy.Name)
			g.Expect(err).NotTo(HaveOccurred())

			evs = testutils.EventFilter(evs, "Warning", "NonCompliant", time.Time{})

			names := make([]string, 0, len(evs))
			for _, ev := range evs {
				if _, ok := ev.Annotations[annoKey]; ok {
					continue
				}

				names = append(names, ev.Name)
			}

			return names
		}, HaveLen(1), policy.GetName) // this policy.GetName is for additional EC coverage
	})

	It("Should emit with the annotation, when eventAnnotation is set", func(ctx SpecContext) {
		Expect(tk.CleanlyCreate(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Name:      "hello-world",
			Namespace: "default",
			Labels:    map[string]string{"sample": ""},
		}})).To(Succeed())

		By("Setting the eventAnnotation field on the policy")
		Expect(tk.Kubectl("patch", "fakepolicy", "-n="+testNS, policy.Name, "--type=json",
			`-p=[{"op": "replace", "path": "/spec/eventAnnotation", "value": "borogoves"}]`,
		)).Should(ContainSubstring("patched"))

		tk.EC(func(g Gomega) []string {
			evs, err := tk.GetComplianceEvents(ctx, testNS, parent.UID, policy.Name)
			g.Expect(err).NotTo(HaveOccurred())

			evs = testutils.EventFilter(evs, "Normal", "^Compliant", time.Time{})

			names := make([]string, 0, len(evs))
			for _, ev := range evs {
				if ev.Annotations[annoKey] != "borogoves" {
					continue
				}

				names = append(names, ev.Name)
			}

			return names
		}, HaveLen(1), "Events counted have annotation '%v'", "borogoves")
	})
})
