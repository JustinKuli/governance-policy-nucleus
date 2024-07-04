// Copyright Contributors to the Open Cluster Management project

package basic

import (
	"fmt"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nucleusv1beta1 "open-cluster-management.io/governance-policy-nucleus/api/v1beta1"
	"open-cluster-management.io/governance-policy-nucleus/pkg/testutils"
	fakev1beta1 "open-cluster-management.io/governance-policy-nucleus/test/fakepolicy/api/v1beta1"
	. "open-cluster-management.io/governance-policy-nucleus/test/fakepolicy/test/utils"
)

var _ = Describe("FakePolicy NamespaceSelection", Ordered, func() {
	defaultNamespaces := []string{"default", "kube-node-lease", "kube-public", "kube-system"}
	sampleNamespaces := []string{"foo", "goo", "fake", "faze", "kube-one"}
	allNamespaces := append(defaultNamespaces, sampleNamespaces...)

	BeforeAll(func(ctx SpecContext) {
		By("Creating sample namespaces")
		for _, ns := range sampleNamespaces {
			nsObj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name:   ns,
				Labels: map[string]string{"sample": ns},
			}}
			Expect(tk.CleanlyCreate(ctx, nsObj)).To(Succeed())
		}

		By("Ensuring the allNamespaces list is correct")
		// constructing the default / allNamespaces lists is complicated because of how ginkgo
		// runs the table tests... this seems better than other workarounds.
		nsList := corev1.NamespaceList{}
		Expect(tk.List(ctx, &nsList)).To(Succeed())

		foundNS := make([]string, len(nsList.Items))
		for i, ns := range nsList.Items {
			foundNS[i] = ns.GetName()
		}

		Expect(allNamespaces).To(ConsistOf(foundNS))
	})

	DescribeTable("Verifying NamespaceSelector behavior",
		func(ctx SpecContext, sel nucleusv1beta1.NamespaceSelector, desiredMatches []string, selErr string) {
			policy := SampleFakePolicy()
			policy.Spec.NamespaceSelector = sel

			Expect(tk.CleanlyCreate(ctx, &policy)).To(Succeed())

			slices.Sort(desiredMatches)

			Eventually(func(g Gomega) {
				foundPolicy := fakev1beta1.FakePolicy{}
				g.Expect(tk.Get(ctx, testutils.ObjNN(&policy), &foundPolicy)).To(Succeed())
				g.Expect(foundPolicy.Status.SelectionComplete).To(BeTrue())

				idx, cond := foundPolicy.Status.GetCondition("NamespaceSelection")
				g.Expect(idx).NotTo(Equal(-1))
				if selErr != "" {
					g.Expect(cond.Message).To(Equal(selErr))
				} else {
					g.Expect(cond.Message).To(Equal(fmt.Sprintf("%v", desiredMatches)))
				}
			}).Should(Succeed())
		},

		// Basic testing of includes and excludes
		Entry("include all with *", nucleusv1beta1.NamespaceSelector{
			Include: []nucleusv1beta1.NonEmptyString{"*"},
		}, allNamespaces, ""),
		Entry("include a specific namespace", nucleusv1beta1.NamespaceSelector{
			Include: []nucleusv1beta1.NonEmptyString{"foo"},
		}, []string{"foo"}, ""),
		Entry("include multiple namespaces with a wildcard", nucleusv1beta1.NamespaceSelector{
			Include: []nucleusv1beta1.NonEmptyString{"fa?e"},
		}, []string{"fake", "faze"}, ""),
		Entry("exclude namespaces with a wildcard", nucleusv1beta1.NamespaceSelector{
			Include: []nucleusv1beta1.NonEmptyString{"*"},
			Exclude: []nucleusv1beta1.NonEmptyString{"kube-*"},
		}, []string{"default", "foo", "goo", "fake", "faze"}, ""),
		Entry("error if an include entry is malformed", nucleusv1beta1.NamespaceSelector{
			Include: []nucleusv1beta1.NonEmptyString{"kube-[system"},
		}, []string{}, "error parsing 'include' pattern 'kube-[system': syntax error in pattern"),

		// Testing with label selector
		Entry("select by a label existing", nucleusv1beta1.NamespaceSelector{
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "sample",
					Operator: metav1.LabelSelectorOpExists,
				}},
			},
		}, sampleNamespaces, ""),
		Entry("select by a label matching a specific value", nucleusv1beta1.NamespaceSelector{
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"sample": "foo",
				},
			},
		}, []string{"foo"}, ""),
		Entry("select using a label and an expression", nucleusv1beta1.NamespaceSelector{
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kubernetes.io/metadata.name": "default",
				},
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "sample",
					Operator: metav1.LabelSelectorOpDoesNotExist,
				}},
			},
		}, []string{"default"}, ""),
		Entry("include a subset with a label existing", nucleusv1beta1.NamespaceSelector{
			Include: []nucleusv1beta1.NonEmptyString{"foo", "goo"},
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "sample",
					Operator: metav1.LabelSelectorOpExists,
				}},
			},
		}, []string{"foo", "goo"}, ""),
		Entry("exclude a subset with a label existing", nucleusv1beta1.NamespaceSelector{
			Exclude: []nucleusv1beta1.NonEmptyString{"f*"},
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "sample",
					Operator: metav1.LabelSelectorOpExists,
				}},
			},
		}, []string{"goo", "kube-one"}, ""),
		Entry("error if the LabelSelector is malformed", nucleusv1beta1.NamespaceSelector{
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "sample",
					Operator: metav1.LabelSelectorOpExists,
					Values:   []string{"foo"},
				}},
			},
		}, []string{}, "values: Invalid value: []string{\"foo\"}: "+
			"values set must be empty for exists and does not exist"),

		// Various flavors of "nil" - when left unset, or specifically set to nil
		Entry("all nil fields", nucleusv1beta1.NamespaceSelector{
			LabelSelector: &metav1.LabelSelector{
				MatchLabels:      nil,
				MatchExpressions: nil,
			},
			Include: nil,
			Exclude: nil,
		}, []string{}, ""),
		Entry("empty LabelSelector", nucleusv1beta1.NamespaceSelector{
			// because of go's zero values, this is exactly the same as 'all nil fields'
			LabelSelector: &metav1.LabelSelector{},
			Include:       nil,
			Exclude:       nil,
		}, []string{}, ""),
		Entry("include, exclude, and selector all nil", nucleusv1beta1.NamespaceSelector{
			LabelSelector: nil,
			Include:       nil,
			Exclude:       nil,
		}, []string{}, ""),
		Entry("empty Target", nucleusv1beta1.NamespaceSelector{}, []string{}, ""),

		// When the LabelSelector is specified, it should be used
		Entry("all empty initialized fields in the Target", nucleusv1beta1.NamespaceSelector{
			LabelSelector: &metav1.LabelSelector{
				MatchLabels:      map[string]string{},
				MatchExpressions: []metav1.LabelSelectorRequirement{},
			},
			Include: []nucleusv1beta1.NonEmptyString{},
			Exclude: []nucleusv1beta1.NonEmptyString{},
		}, allNamespaces, ""),
		Entry("specified empty MatchLabels", nucleusv1beta1.NamespaceSelector{
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{},
			},
			Include: []nucleusv1beta1.NonEmptyString{},
			Exclude: []nucleusv1beta1.NonEmptyString{},
		}, allNamespaces, ""),
		Entry("specified empty MatchExpressions", nucleusv1beta1.NamespaceSelector{
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{},
			},
			Include: []nucleusv1beta1.NonEmptyString{},
			Exclude: []nucleusv1beta1.NonEmptyString{},
		}, allNamespaces, ""),

		// Interactions between the various kinds of "empty" LabelSelector and Include.
		Entry("nil fields in the LabelSelector", nucleusv1beta1.NamespaceSelector{
			LabelSelector: &metav1.LabelSelector{
				MatchLabels:      nil,
				MatchExpressions: nil,
			},
			Include: []nucleusv1beta1.NonEmptyString{"foo"},
		}, []string{"foo"}, ""),
		Entry("empty LabelSelector", nucleusv1beta1.NamespaceSelector{
			LabelSelector: &metav1.LabelSelector{},
			Include:       []nucleusv1beta1.NonEmptyString{"foo"},
		}, []string{"foo"}, ""),
		Entry("nil LabelSelector", nucleusv1beta1.NamespaceSelector{
			LabelSelector: nil,
			Include:       []nucleusv1beta1.NonEmptyString{"foo"},
		}, []string{"foo"}, ""),
		Entry("initialized empty fields inside the LabelSelector", nucleusv1beta1.NamespaceSelector{
			LabelSelector: &metav1.LabelSelector{
				MatchLabels:      map[string]string{},
				MatchExpressions: []metav1.LabelSelectorRequirement{},
			},
			Include: []nucleusv1beta1.NonEmptyString{"foo"},
		}, []string{"foo"}, ""),
	)
})
