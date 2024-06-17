// Copyright Contributors to the Open Cluster Management project

package compliance

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"open-cluster-management.io/governance-policy-nucleus/pkg/testutils"
	fakev1beta1 "open-cluster-management.io/governance-policy-nucleus/test/fakepolicy/api/v1beta1"
	. "open-cluster-management.io/governance-policy-nucleus/test/fakepolicy/test/utils"
)

var _ = Describe("Classic Compliance Events", Ordered, func() {
	const testNS string = "classic-comp-test"

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
		policy.Name += "-classic-comp-test"
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

	It("Should emit one NonCompliant event", func(ctx SpecContext) {
		tk.EC(func(g Gomega) []string {
			evs, err := tk.GetComplianceEvents(ctx, testNS, parent.UID, policy.Name)
			g.Expect(err).NotTo(HaveOccurred())

			evs = testutils.EventFilter(evs, "Warning", "NonCompliant", time.Time{})

			names := make([]string, 0, len(evs))
			for _, ev := range evs {
				names = append(names, ev.Name)
			}

			return names
		}, HaveLen(1), "Only counting NonCompliant events")
	})

	It("Should emit one Compliant event after the configmap is created", func(ctx SpecContext) {
		Expect(tk.CleanlyCreate(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Name:      "hello-world",
			Namespace: "default",
			Labels:    map[string]string{"sample": ""},
		}})).To(Succeed())

		// Refresh the test's copy of the policy
		Expect(tk.Get(ctx, testutils.ObjNN(policy), policy)).To(Succeed())

		By("Setting an annotation on the policy to trigger a re-reconcile")
		policy.SetAnnotations(map[string]string{
			"classic-comp-test-1": "1",
		})
		Expect(tk.Update(ctx, policy)).To(Succeed())

		tk.EC(func(g Gomega) []string {
			evs, err := tk.GetComplianceEvents(ctx, testNS, parent.UID, policy.Name)
			g.Expect(err).NotTo(HaveOccurred())

			evs = testutils.EventFilter(evs, "Normal", "^Compliant", time.Time{})

			names := make([]string, 0, len(evs))
			for _, ev := range evs {
				names = append(names, ev.Name)
			}

			return names
		}, HaveLen(1), "Only counting Compliant events")
	})

	It("Should emit a NonCompliant event after the configmap is deleted", func(ctx SpecContext) {
		By("Ensuring that the configmap is gone")
		Eventually(func() string {
			cm := corev1.ConfigMap{}
			_ = tk.Get(ctx, types.NamespacedName{Name: "hello-world", Namespace: "default"}, &cm)

			return cm.Name
		}, "1s", "50ms").Should(BeEmpty())

		// Refresh the test's copy of the policy
		Expect(tk.Get(ctx, testutils.ObjNN(policy), policy)).To(Succeed())

		By("Patching an annotation on the policy to trigger a re-reconcile")
		patch := `[{"op":"replace","path":"/metadata/annotations/classic-comp-test-1","value":"2"}]`
		err := tk.Patch(ctx, policy, client.RawPatch(types.JSONPatchType, []byte(patch)))
		Expect(err).NotTo(HaveOccurred())

		// This is just an example usage of this function, not an actual case where it was necessary.
		// It's _obvious_ (/s) that the test must use a filter for 2 seconds ago; if it filtered to
		// only 1 second ago, then since the Consistently runs for a whole second, it would always
		// get an empty list near the end, and fail.
		debugMsg := testutils.RegisterDebugMessage()

		tk.EC(func(g Gomega) []string {
			evs, err := tk.GetComplianceEvents(ctx, testNS, parent.UID, policy.Name)
			g.Expect(err).NotTo(HaveOccurred())

			*debugMsg = "unfiltered events: "
			for _, ev := range evs {
				*debugMsg += fmt.Sprintf("(%v: %v), ", ev.Name, ev.Message)
			}

			evs = testutils.EventFilter(evs, "Warning", "NonCompliant",
				time.Now().Add(-2*time.Second))

			names := make([]string, 0, len(evs))
			for _, ev := range evs {
				names = append(names, ev.Name)
			}

			return names
		}, HaveLen(1), "Only counting NonCompliant events from the last 2 seconds")
	})
})
