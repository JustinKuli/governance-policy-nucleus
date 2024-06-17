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
func (e K8sEmitter) EmitEvent(ctx context.Context, pl nucleusv1beta1.PolicyLike) (*corev1.Event, error) {
	plGVK := pl.GetObjectKind().GroupVersionKind()
	time := time.Now()

	// This event name matches the convention of recorders from client-go
	name := fmt.Sprintf("%v.%x", pl.Parent().Name, time.UnixNano())

	// The reason must match a pattern looked for by the policy framework
	var reason string
	if ns := pl.GetNamespace(); ns != "" {
		reason = "policy: " + ns + "/" + pl.GetName()
	} else {
		reason = "policy: " + pl.GetName()
	}

	// The message must begin with the compliance, then should go into a descriptive message
	message := string(pl.ComplianceState()) + "; " + pl.ComplianceMessage()

	evType := "Normal"
	if pl.ComplianceState() != nucleusv1beta1.Compliant {
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
			Namespace:   pl.ParentNamespace(),
			Labels:      pl.GetLabels(),
			Annotations: pl.GetAnnotations(),
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:       pl.Parent().Kind,
			Namespace:  pl.ParentNamespace(),
			Name:       pl.Parent().Name,
			UID:        pl.Parent().UID,
			APIVersion: pl.Parent().APIVersion,
		},
		Reason:         reason,
		Message:        message,
		Source:         src,
		FirstTimestamp: metav1.NewTime(time),
		LastTimestamp:  metav1.NewTime(time),
		Count:          1,
		Type:           evType,
		EventTime:      metav1.NewMicroTime(time),
		Series:         nil,
		Action:         "ComplianceStateUpdate",
		Related: &corev1.ObjectReference{
			Kind:            plGVK.Kind,
			Namespace:       pl.GetNamespace(),
			Name:            pl.GetName(),
			UID:             pl.GetUID(),
			APIVersion:      plGVK.GroupVersion().String(),
			ResourceVersion: pl.GetResourceVersion(),
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
