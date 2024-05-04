// Copyright Contributors to the Open Cluster Management project

package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nucleusv1alpha1 "open-cluster-management.io/governance-policy-nucleus/api/v1alpha1"
	nucleusv1beta1 "open-cluster-management.io/governance-policy-nucleus/api/v1beta1"
	fakev1beta1 "open-cluster-management.io/governance-policy-nucleus/test/fakepolicy/api/v1beta1"
)

// FakePolicyReconciler reconciles a FakePolicy object
type FakePolicyReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	DynamicClient *dynamic.DynamicClient
}

// Usual RBAC for fakepolicy:
//+kubebuilder:rbac:groups=policy.open-cluster-management.io,resources=fakepolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=policy.open-cluster-management.io,resources=fakepolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=policy.open-cluster-management.io,resources=fakepolicies/finalizers,verbs=update

// Nucleus RBAC for namespaceSelector:
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch

// RBAC for this fakepolicy's capabilities:
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch

func (r *FakePolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	policy := &fakev1beta1.FakePolicy{}
	if err := r.Get(ctx, req.NamespacedName, policy); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, probably deleted
			return ctrl.Result{}, nil
		}

		log.Error(err, "Failed to get FakePolicy")

		return ctrl.Result{}, err
	}

	policy.Status.SelectionError = ""

	selectedNamespaces, err := policy.Spec.NamespaceSelector.GetNamespaces(ctx, r.Client)
	if err != nil {
		log.Error(err, "Failed to GetNamespaces using NamespaceSelector",
			"selector", policy.Spec.NamespaceSelector)

		policy.Status.SelectionError = err.Error()
	}

	policy.Status.SelectedNamespaces = selectedNamespaces

	cmIface := r.DynamicClient.Resource(schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	})

	dynamicMatchedCMs, err := policy.Spec.TargetConfigMaps.GetMatchesDynamic(ctx, cmIface)
	if err != nil {
		log.Error(err, "Failed to GetMatchesDynamic for the TargetConfigMaps",
			"target", policy.Spec.TargetConfigMaps)

		policy.Status.SelectionError = err.Error()
	}

	policy.Status.DynamicSelectedConfigMaps = make([]string, len(dynamicMatchedCMs))
	for i, cm := range dynamicMatchedCMs {
		policy.Status.DynamicSelectedConfigMaps[i] = cm.GetNamespace() + "/" + cm.GetName()
	}

	var list nucleusv1beta1.ResourceList

	if policy.Spec.TargetUsingReflection {
		list = &nucleusv1alpha1.ReflectiveResourceList{ClientList: &corev1.ConfigMapList{}}
	} else {
		list = &configMapResList{}
	}

	clientMatchedCMs, err := policy.Spec.TargetConfigMaps.GetMatches(ctx, r.Client, list)
	if err != nil {
		log.Error(err, "Failed to GetMatches for the TargetConfigMaps",
			"target", policy.Spec.TargetConfigMaps)

		policy.Status.SelectionError = err.Error()
	}

	policy.Status.ClientSelectedConfigMaps = make([]string, len(clientMatchedCMs))
	for i, cm := range clientMatchedCMs {
		policy.Status.ClientSelectedConfigMaps[i] = cm.GetNamespace() + "/" + cm.GetName()
	}

	policy.Status.SelectionComplete = true

	if err := r.Status().Update(ctx, policy); err != nil {
		log.Error(err, "Failed to update status")

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

type configMapResList struct {
	corev1.ConfigMapList
}

// ensure configMapResList implements ResourceList
var _ nucleusv1beta1.ResourceList = (*configMapResList)(nil)

func (l *configMapResList) Items() ([]client.Object, error) {
	items := make([]client.Object, len(l.ConfigMapList.Items))
	for i := range l.ConfigMapList.Items {
		items[i] = &l.ConfigMapList.Items[i]
	}

	return items, nil
}

func (l *configMapResList) ObjectList() client.ObjectList {
	return &l.ConfigMapList
}

// SetupWithManager sets up the controller with the Manager.
func (r *FakePolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&fakev1beta1.FakePolicy{}).
		Complete(r)
}
