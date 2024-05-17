// Copyright Contributors to the Open Cluster Management project

package basic

import (
	"fmt"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nucleusv1beta1 "open-cluster-management.io/governance-policy-nucleus/api/v1beta1"
	"open-cluster-management.io/governance-policy-nucleus/pkg/testutils"
	fakev1beta1 "open-cluster-management.io/governance-policy-nucleus/test/fakepolicy/api/v1beta1"
	. "open-cluster-management.io/governance-policy-nucleus/test/fakepolicy/test/utils"
)

var _ = Describe("FakePolicy TargetConfigMaps", func() {
	defaultConfigMaps := []string{
		"kube-system/extension-apiserver-authentication",
		"kube-system/kube-apiserver-legacy-service-account-token-tracking",
	}
	sampleConfigMaps := []string{
		"default/foo",
		"default/goo",
		"default/fake",
		"default/faze",
		"default/kube-one",
		"default/extension-apiserver-authentication",
		"kube-public/kube-testing",
	}
	allConfigMaps := append(defaultConfigMaps, sampleConfigMaps...)

	beforeFunc := func(ctx SpecContext) {
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
			Expect(tk.CleanlyCreate(ctx, cmObj)).To(Succeed())
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
	}

	entries := []TableEntry{
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
		}, []string{
			"default/goo",
			"default/kube-one",
			"default/extension-apiserver-authentication",
			"kube-public/kube-testing",
		}, ""),
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
	}

	checkFunc := func(policy fakev1beta1.FakePolicy, desiredMatches []string, selErr string) func(g Gomega) {
		return func(g Gomega) {
			foundPolicy := fakev1beta1.FakePolicy{}
			g.Expect(k8sClient.Get(ctx, testutils.ObjNN(&policy), &foundPolicy)).To(Succeed())
			g.Expect(foundPolicy.Status.SelectionComplete).To(BeTrue())

			slices.Sort(desiredMatches)

			idx, cond := foundPolicy.Status.GetCondition("DynamicSelection")
			g.Expect(idx).NotTo(Equal(-1))
			if selErr != "" {
				g.Expect(cond.Message).To(Equal(selErr))
			} else {
				g.Expect(cond.Message).To(Equal(fmt.Sprintf("%v", desiredMatches)))
			}

			idx, cond = foundPolicy.Status.GetCondition("ClientSelection")
			g.Expect(idx).NotTo(Equal(-1))
			if selErr != "" {
				g.Expect(cond.Message).To(Equal(selErr))
			} else {
				g.Expect(cond.Message).To(Equal(fmt.Sprintf("%v", desiredMatches)))
			}
		}
	}

	Describe("Targets without a Namespace", Ordered, func() {
		BeforeAll(beforeFunc)

		DescribeTable("Verifying TargetConfigMaps behavior",
			func(ctx SpecContext, sel nucleusv1beta1.Target, desiredMatches []string, selErr string) {
				policy := SampleFakePolicy()
				policy.Spec.TargetConfigMaps = sel

				Expect(tk.CleanlyCreate(ctx, &policy)).To(Succeed())

				Eventually(checkFunc(policy, desiredMatches, selErr)).Should(Succeed())
			},
			entries,
		)
	})

	Describe("Targets restricted to the default namespace", Ordered, func() {
		BeforeAll(beforeFunc)

		DescribeTable("Verifying TargetConfigMaps behavior",
			func(ctx SpecContext, sel nucleusv1beta1.Target, givenDesiredMatches []string, selErr string) {
				sel.Namespace = "default"

				desiredMatches := make([]string, 0)
				for _, item := range givenDesiredMatches {
					ns, _, _ := strings.Cut(item, "/")
					if ns == "default" {
						desiredMatches = append(desiredMatches, item)
					}
				}

				policy := SampleFakePolicy()
				policy.Spec.TargetConfigMaps = sel

				Expect(tk.CleanlyCreate(ctx, &policy)).To(Succeed())

				Eventually(checkFunc(policy, desiredMatches, selErr)).Should(Succeed())
			},
			entries,
		)
	})

	Describe("Targets using ReflectiveResourceList", Ordered, func() {
		BeforeAll(beforeFunc)

		DescribeTable("Verifying TargetConfigMaps behavior",
			func(ctx SpecContext, sel nucleusv1beta1.Target, desiredMatches []string, selErr string) {
				policy := SampleFakePolicy()
				policy.Spec.TargetConfigMaps = sel
				policy.Spec.TargetUsingReflection = true

				Expect(tk.CleanlyCreate(ctx, &policy)).To(Succeed())

				Eventually(checkFunc(policy, desiredMatches, selErr)).Should(Succeed())
			},
			entries,
		)
	})
})
