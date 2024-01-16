// Copyright Contributors to the Open Cluster Management project

package v1beta1

import (
	"context"
	"fmt"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetNamespaces fetches all namespaces in the cluster and returns a list of the
// namespaces that match the NamespaceSelector. The client.Reader needs access
// for viewing namespaces, like the access given by this kubebuilder tag:
// `//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch`
func (sel NamespaceSelector) GetNamespaces(ctx context.Context, r client.Reader) ([]string, error) {
	if len(sel.Include) == 0 && sel.LabelSelector == nil {
		// A somewhat special case of no matches.
		return []string{}, nil
	}

	listOpts := client.ListOptions{}

	if sel.LabelSelector != nil {
		labelSel, err := metav1.LabelSelectorAsSelector(sel.LabelSelector)
		if err != nil {
			return nil, err
		}

		listOpts.LabelSelector = labelSel
	}

	namespaceList := &corev1.NamespaceList{}
	if err := r.List(ctx, namespaceList, &listOpts); err != nil {
		return nil, err
	}

	namespaces := make([]string, len(namespaceList.Items))
	for i, ns := range namespaceList.Items {
		namespaces[i] = ns.GetName()
	}

	return Target(sel).matches(namespaces)
}

type Target struct {
	*metav1.LabelSelector `json:",inline"`

	// Include is a list of filepath expressions to include objects by name.
	Include []NonEmptyString `json:"include,omitempty"`

	// Exclude is a list of filepath expressions to include objects by name.
	Exclude []NonEmptyString `json:"exclude,omitempty"`
}

// GetMatchesDynamic returns a list of resources on the cluster, matched by the Target. The kind of
// the resources, and whether the list is from one namespace or all namespaces, is configured by the
// input dynamic.ResourceInterface. NOTE: unlike the NamespaceSelector, an empty Target will match
// *all* resources on the cluster.
func (t Target) GetMatchesDynamic(ctx context.Context, iface dynamic.ResourceInterface,
) ([]*unstructured.Unstructured, error) {
	labelSel, err := metav1.LabelSelectorAsSelector(t.LabelSelector)
	if err != nil {
		return nil, err
	}

	objs, err := iface.List(ctx, metav1.ListOptions{LabelSelector: labelSel.String()})
	if err != nil {
		return nil, err
	}

	matchedObjs := make([]*unstructured.Unstructured, 0)

	for _, obj := range objs.Items {
		obj := obj

		matched, err := t.match(obj.GetName())
		if err != nil {
			return matchedObjs, err
		}

		if matched {
			matchedObjs = append(matchedObjs, &obj)
		}
	}

	return matchedObjs, nil
}

// matches filters a slice of strings, and returns ones that match the Include
// and Exclude lists in the Target. The only possible returned error is a
// wrapped filepath.ErrBadPattern.
func (t Target) matches(names []string) ([]string, error) {
	// Using a map to ensure each entry in the result is unique.
	set := make(map[string]struct{})

	for _, name := range names {
		matched, err := t.match(name)
		if err != nil {
			return nil, err
		}

		if matched {
			set[name] = struct{}{}
		}
	}

	matchingNames := make([]string, 0, len(set))
	for ns := range set {
		matchingNames = append(matchingNames, ns)
	}

	return matchingNames, nil
}

// match returns whether the given name matches the Include and Exclude lists in
// the Target.
func (t Target) match(name string) (bool, error) {
	var err error

	include := len(t.Include) == 0 // include everything if empty/unset

	for _, includePattern := range t.Include {
		include, err = filepath.Match(string(includePattern), name)
		if err != nil {
			return false, fmt.Errorf("error parsing 'include' pattern '%s': %w", string(includePattern), err)
		}

		if include {
			break
		}
	}

	if !include {
		return false, nil
	}

	for _, excludePattern := range t.Exclude {
		exclude, err := filepath.Match(string(excludePattern), name)
		if err != nil {
			return false, fmt.Errorf("error parsing 'exclude' pattern '%s': %w", string(excludePattern), err)
		}

		if exclude {
			return false, nil
		}
	}

	return true, nil
}
