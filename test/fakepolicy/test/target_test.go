// Copyright Contributors to the Open Cluster Management project

package test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nucleusv1beta1 "open-cluster-management.io/governance-policy-nucleus/api/v1beta1"
	fakev1beta1 "open-cluster-management.io/governance-policy-nucleus/test/fakepolicy/api/v1beta1"
)

var _ = Describe("FakePolicy TargetConfigMaps", Ordered, func() {
	defaultConfigMaps := []string{"kube-system/extension-apiserver-authentication"}
	sampleConfigMaps := []string{
		"default/foo",
		"default/goo",
		"default/fake",
		"default/faze",
		"default/kube-one",
		"default/extension-apiserver-authentication",
	}
	allConfigMaps := append(defaultConfigMaps, sampleConfigMaps...)

	BeforeAll(func() {
		By("Creating sample configmaps")
		for _, cm := range sampleConfigMaps {
			ns, name, _ := strings.Cut(cm, "/")
			cmObj := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: ns,
					Labels:    map[string]string{"sample": name},
				},
				Data: map[string]string{"foo": "bar"},
			}
			Expect(cleanlyCreate(cmObj)).To(Succeed())
		}

		By("Ensuring the allConfigMaps list is correct")
		// constructing the default / allConfigMaps lists is complicated because of how ginkgo
		// runs the table tests... this seems better than other workarounds.
		cmList := corev1.ConfigMapList{}
		Expect(k8sClient.List(ctx, &cmList)).To(Succeed())

		foundCM := make([]string, len(cmList.Items))
		for i, cm := range cmList.Items {
			foundCM[i] = cm.GetNamespace() + "/" + cm.GetName()
		}

		Expect(allConfigMaps).To(ConsistOf(foundCM))
	})

	DescribeTable("Verifying TargetConfigMaps behavior",
		func(sel nucleusv1beta1.Target, desiredMatches []string, selErr string) {
			policy := sampleFakePolicy()
			policy.Spec.TargetConfigMaps = sel

			Expect(cleanlyCreate(&policy)).To(Succeed())

			Eventually(func(g Gomega) {
				foundPolicy := fakev1beta1.FakePolicy{}
				g.Expect(k8sClient.Get(ctx, getNamespacedName(&policy), &foundPolicy)).To(Succeed())
				g.Expect(foundPolicy.Status.SelectionComplete).To(BeTrue())
				g.Expect(foundPolicy.Status.DynamicSelectedConfigMaps).To(ConsistOf(desiredMatches))
				g.Expect(foundPolicy.Status.ClientSelectedConfigMaps).To(ConsistOf(desiredMatches))
				g.Expect(foundPolicy.Status.SelectionError).To(Equal(selErr))
			}).Should(Succeed())
		},
		//
		Entry("empty Target", nucleusv1beta1.Target{}, allConfigMaps, ""),

		// Basic testing of includes and excludes
		Entry("include all with *", nucleusv1beta1.Target{
			Include: []nucleusv1beta1.NonEmptyString{"*"},
		}, allConfigMaps, ""),
		Entry("include a specific configmap", nucleusv1beta1.Target{
			Include: []nucleusv1beta1.NonEmptyString{"foo"},
		}, []string{"default/foo"}, ""),
		Entry("include multiple configmaps with a wildcard", nucleusv1beta1.Target{
			Include: []nucleusv1beta1.NonEmptyString{"fa?e"},
		}, []string{"default/fake", "default/faze"}, ""),
		Entry("exclude configmaps with wildcards", nucleusv1beta1.Target{
			Include: []nucleusv1beta1.NonEmptyString{"*"},
			Exclude: []nucleusv1beta1.NonEmptyString{"kube-*", "extension-*"},
		}, []string{"default/foo", "default/goo", "default/fake", "default/faze"}, ""),
		Entry("error if an include entry is malformed", nucleusv1beta1.Target{
			Include: []nucleusv1beta1.NonEmptyString{"kube-[system"},
		}, []string{}, "error parsing 'include' pattern 'kube-[system': syntax error in pattern"),

		// Testing with label selector
		Entry("select by a label existing", nucleusv1beta1.Target{
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "sample",
					Operator: metav1.LabelSelectorOpExists,
				}},
			},
		}, sampleConfigMaps, ""),
		Entry("select by a label matching a specific value", nucleusv1beta1.Target{
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"sample": "foo",
				},
			},
		}, []string{"default/foo"}, ""),
		Entry("include a subset with a label existing", nucleusv1beta1.Target{
			Include: []nucleusv1beta1.NonEmptyString{"foo", "goo"},
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "sample",
					Operator: metav1.LabelSelectorOpExists,
				}},
			},
		}, []string{"default/foo", "default/goo"}, ""),
		Entry("exclude a subset with a label existing", nucleusv1beta1.Target{
			Exclude: []nucleusv1beta1.NonEmptyString{"f*"},
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "sample",
					Operator: metav1.LabelSelectorOpExists,
				}},
			},
		}, []string{"default/goo", "default/kube-one", "default/extension-apiserver-authentication"}, ""),
		Entry("error if the LabelSelector is malformed", nucleusv1beta1.Target{
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "sample",
					Operator: metav1.LabelSelectorOpExists,
					Values:   []string{"foo"},
				}},
			},
		}, []string{}, "values: Invalid value: []string{\"foo\"}: "+
			"values set must be empty for exists and does not exist"),
	)
})
