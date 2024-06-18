// Copyright Contributors to the Open Cluster Management project

package compliance

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nucleusv1beta1 "open-cluster-management.io/governance-policy-nucleus/api/v1beta1"
)

// K8sEmitter is an emitter of Kubernetes events which the policy framework
// watches for in order to aggregate and report policy status.
type K8sEmitter struct {
	// Client is a Kubernetes client for the cluster where the compliance events
	// will be created.
	Client client.Client

	// Source contains optional information for where the event comes from.
	Source corev1.EventSource

	// Mutators modify the Event after the fields are initially set, but before
	// it is created on the cluster. They are run in the order they are defined.
	Mutators []func(corev1.Event) (corev1.Event, error)
}

// Emit creates the Kubernetes Event on the cluster. It returns an error if the
// API call fails.
func (e K8sEmitter) Emit(ctx context.Context, pl nucleusv1beta1.PolicyLike) error {
	_, err := e.EmitEvent(ctx, pl)

	return err
}

// EmitEvent creates the Kubernetes Event on the cluster. It returns the Event
// that was (at least) attempted to be created, and an error if the API call
// fails.
func (e K8sEmitter) EmitEvent(ctx context.Context, pol nucleusv1beta1.PolicyLike) (*corev1.Event, error) {
	plGVK := pol.GetObjectKind().GroupVersionKind()
	now := time.Now()

	// This event name matches the convention of recorders from client-go
	name := fmt.Sprintf("%v.%x", pol.Parent().Name, now.UnixNano())

	// The reason must match a pattern looked for by the policy framework
	var reason string
	if ns := pol.GetNamespace(); ns != "" {
		reason = "policy: " + ns + "/" + pol.GetName()
	} else {
		reason = "policy: " + pol.GetName()
	}

	// The message must begin with the compliance, then should go into a descriptive message
	message := string(pol.ComplianceState()) + "; " + pol.ComplianceMessage()

	evType := "Normal"
	if pol.ComplianceState() != nucleusv1beta1.Compliant {
		evType = "Warning"
	}

	src := corev1.EventSource{
		Component: e.Source.Component,
		Host:      e.Source.Host,
	}

	// These fields are required for the event to function as expected
	if src.Component == "" {
		src.Component = "policy-nucleus-default"
	}

	if src.Host == "" {
		src.Host = "policy-nucleus-default"
	}

	event := corev1.Event{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Event",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   pol.ParentNamespace(),
			Labels:      pol.GetLabels(),
			Annotations: pol.GetAnnotations(),
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:       pol.Parent().Kind,
			Namespace:  pol.ParentNamespace(),
			Name:       pol.Parent().Name,
			UID:        pol.Parent().UID,
			APIVersion: pol.Parent().APIVersion,
		},
		Reason:         reason,
		Message:        message,
		Source:         src,
		FirstTimestamp: metav1.NewTime(now),
		LastTimestamp:  metav1.NewTime(now),
		Count:          1,
		Type:           evType,
		EventTime:      metav1.NewMicroTime(now),
		Series:         nil,
		Action:         "ComplianceStateUpdate",
		Related: &corev1.ObjectReference{
			Kind:            plGVK.Kind,
			Namespace:       pol.GetNamespace(),
			Name:            pol.GetName(),
			UID:             pol.GetUID(),
			APIVersion:      plGVK.GroupVersion().String(),
			ResourceVersion: pol.GetResourceVersion(),
		},
		ReportingController: src.Component,
		ReportingInstance:   src.Host,
	}

	for _, mutatorFunc := range e.Mutators {
		var err error

		event, err = mutatorFunc(event)
		if err != nil {
			return nil, err
		}
	}

	err := e.Client.Create(ctx, &event)

	return &event, err
}
