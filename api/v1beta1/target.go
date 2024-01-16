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

	t := Target{
		LabelSelector: sel.LabelSelector,
		Include:       sel.Include,
		Exclude:       sel.Exclude,
	}

	return t.matches(namespaces)
}

type Target struct {
	*metav1.LabelSelector `json:",inline"`

	// Include is a list of filepath expressions to include objects by name.
	Include []NonEmptyString `json:"include,omitempty"`

	// Exclude is a list of filepath expressions to include objects by name.
	Exclude []NonEmptyString `json:"exclude,omitempty"`

	// Namespace is the namespace to restrict the Target to. Can be empty for non-namespaced
	// objects, or to look in all namespaces.
	Namespace string `json:"namespace,omitempty"`
}

//+kubebuilder:object:generate=false

// ResourceList is meant to wrap a concrete implementation of a client.ObjectList, giving access
// to the items in the list. The methods should be implemented on pointer types.
type ResourceList interface {
	ObjectList() client.ObjectList
	Items() ([]client.Object, error)
}

// GetMatches returns a list of resources on the cluster, matched by the Target. The provided
// ResourceList should be backed by a client.ObjectList type which must registered in the scheme of
// the client.Reader. The items in the provided ResourceList after this method is called will not
// necessarily equal the items matched by the Target.
//
// This method should be used preferentially to `GetMatchesDynamic` because it can leverage the
// Reader's cache. NOTE: unlike the NamespaceSelector, an empty Target will match *all* resources on
// the cluster.
func (t Target) GetMatches(ctx context.Context, r client.Reader, list ResourceList) ([]client.Object, error) {
	nonNilSel := t.LabelSelector
	if nonNilSel == nil { // override it to be empty if it is nil
		nonNilSel = &metav1.LabelSelector{}
	}

	labelSel, err := metav1.LabelSelectorAsSelector(nonNilSel)
	if err != nil {
		return nil, err
	}

	listOpts := client.ListOptions{
		LabelSelector: labelSel,
		Namespace:     t.Namespace,
	}

	if err := r.List(ctx, list.ObjectList(), &listOpts); err != nil {
		return nil, err
	}

	items, err := list.Items()
	if err != nil {
		return nil, err
	}

	return t.matchesByName(items)
}

// GetMatchesDynamic returns a list of resources on the cluster, matched by the Target. The kind of
// the resources is configured by the provided dynamic.ResourceInterface. If the Target specifies a
// namespace, this method will limit the namespace of the provided Interface if possible. If the
// provided Interface is already namespaced, the namespace of the Interface will be used (possibly
// overriding the namespace specified in the Target).
//
// NOTE: unlike the NamespaceSelector, an empty Target will match *all* resources on the cluster.
func (t Target) GetMatchesDynamic(ctx context.Context, iface dynamic.ResourceInterface,
) ([]*unstructured.Unstructured, error) {
	labelSel, err := metav1.LabelSelectorAsSelector(t.LabelSelector)
	if err != nil {
		return nil, err
	}

	if t.Namespace != "" {
		if namespaceableIface, ok := iface.(dynamic.NamespaceableResourceInterface); ok {
			iface = namespaceableIface.Namespace(t.Namespace)
		}
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

// matchesByName filters a list of client.Objects by name, and returns ones that
// match the Include and Exclude lists in the Target. The only possible returned
// error is a wrapped filepath.ErrBadPattern.
func (t Target) matchesByName(items []client.Object) ([]client.Object, error) {
	matches := make([]client.Object, 0)

	for _, item := range items {
		matched, err := t.match(item.GetName())
		if err != nil {
			return nil, err
		}

		if matched {
			matches = append(matches, item)
		}
	}

	return matches, nil
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
