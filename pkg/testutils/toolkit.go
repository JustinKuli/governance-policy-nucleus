// Copyright Contributors to the Open Cluster Management project

package testutils

import (
	"context"
	"fmt"
	"regexp"
	"sort"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	gomegaTypes "github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Toolkit struct {
	client.Client
	EventuallyPoll      string
	EventuallyTimeout   string
	ConsistentlyPoll    string
	ConsistentlyTimeout string
	BackgroundCtx       context.Context //nolint:containedctx // this is for convenience
}

// NewToolkit returns a toolkit using the given Client, with some basic defaults.
// This is the preferred way to get a Toolkit instance, to avoid unset fields.
func NewToolkit(client client.Client) Toolkit {
	return Toolkit{
		Client:              client,
		EventuallyPoll:      "100ms",
		EventuallyTimeout:   "1s",
		ConsistentlyPoll:    "100ms",
		ConsistentlyTimeout: "1s",
		BackgroundCtx:       context.Background(),
	}
}

// cleanlyCreate creates the given object, and registers a callback to delete the object which
// Ginkgo will call at the appropriate time. The error from the `Create` call is returned (so it
// can be checked) and the `Delete` callback handles 'NotFound' errors as a success.
func (tk Toolkit) CleanlyCreate(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	createErr := tk.Create(ctx, obj)

	if createErr == nil {
		ginkgo.DeferCleanup(func() {
			ginkgo.GinkgoWriter.Printf("Deleting %v %v/%v\n",
				obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName())

			if err := tk.Delete(tk.BackgroundCtx, obj); err != nil {
				if !errors.IsNotFound(err) {
					// Use Fail in order to provide a custom message with useful information
					ginkgo.Fail(fmt.Sprintf("Expected success or 'NotFound' error, got %v", err), 1)
				}
			}
		})
	}

	return createErr
}

// Create uses the toolkit's client to save the object in the Kubernetes cluster.
// The only change in behavior is that it saves and restores the object's type
// information, which might otherwise be stripped during the API call.
func (tk Toolkit) Create(
	ctx context.Context, obj client.Object, opts ...client.CreateOption,
) error {
	savedGVK := obj.GetObjectKind().GroupVersionKind()
	err := tk.Client.Create(ctx, obj, opts...)
	obj.GetObjectKind().SetGroupVersionKind(savedGVK)

	return err
}

// Patch uses the toolkit's client to patch the object in the Kubernetes cluster.
// The only change in behavior is that it saves and restores the object's type
// information, which might otherwise be stripped during the API call.
func (tk Toolkit) Patch(
	ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption,
) error {
	savedGVK := obj.GetObjectKind().GroupVersionKind()
	err := tk.Client.Patch(ctx, obj, patch, opts...)
	obj.GetObjectKind().SetGroupVersionKind(savedGVK)

	return err
}

// Update uses the toolkit's client to update the object in the Kubernetes cluster.
// The only change in behavior is that it saves and restores the object's type
// information, which might otherwise be stripped during the API call.
func (tk Toolkit) Update(
	ctx context.Context, obj client.Object, opts ...client.UpdateOption,
) error {
	savedGVK := obj.GetObjectKind().GroupVersionKind()
	err := tk.Client.Update(ctx, obj, opts...)
	obj.GetObjectKind().SetGroupVersionKind(savedGVK)

	return err
}

// This regular expression is copied from
// https://github.com/open-cluster-management-io/governance-policy-framework-addon/blob/v0.13.0/controllers/statussync/policy_status_sync.go#L220
var compEventRegex = regexp.MustCompile(`(?i)^policy:\s*(?:([a-z0-9.-]+)\s*\/)?(.+)`)

// GetComplianceEvents queries the cluster and returns a sorted list of the Kubernetes
// compliance events for the given policy.
func (tk Toolkit) GetComplianceEvents(
	ctx context.Context, ns string, parentUID types.UID, templateName string,
) ([]corev1.Event, error) {
	list := &corev1.EventList{}

	err := tk.List(ctx, list, client.InNamespace(ns))
	if err != nil {
		return nil, err
	}

	events := make([]corev1.Event, 0)

	for _, event := range list.Items {
		event := event

		if event.InvolvedObject.UID != parentUID {
			continue
		}

		submatch := compEventRegex.FindStringSubmatch(event.Reason)
		if len(submatch) >= 3 && submatch[2] == templateName {
			events = append(events, event)
		}
	}

	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Name < events[j].Name
	})

	return events, nil
}

// EC runs assertions on asynchronous behavior, both *E*ventually and *C*onsistently,
// using the polling and timeout settings of the toolkit. Its usage should feel familiar
// to gomega users, simply skip the `.Should(...)` call and put your matcher as the second
// parameter here.
func (tk Toolkit) EC(
	actualOrCtx interface{}, matcher gomegaTypes.GomegaMatcher, optionalDescription ...interface{},
) bool {
	ginkgo.GinkgoHelper()

	// Add where the failure occurred to the description
	eDesc := make([]interface{}, 1)
	cDesc := make([]interface{}, 1)

	switch len(optionalDescription) {
	case 0:
		eDesc[0] = "Failed in Eventually"
		cDesc[0] = "Failed in Consistently"
	case 1:
		if origDescFunc, ok := optionalDescription[0].(func() string); ok {
			eDesc[0] = func() string {
				return "Failed in Eventually; " + origDescFunc()
			}
			cDesc[0] = func() string {
				return "Failed in Consistently; " + origDescFunc()
			}
		} else {
			eDesc[0] = "Failed in Eventually; " + optionalDescription[0].(string)
			cDesc[0] = "Failed in Consistently; " + optionalDescription[0].(string)
		}
	default:
		eDesc[0] = "Failed in Eventually; " + optionalDescription[0].(string)
		eDesc = append(eDesc, optionalDescription[1:]...) //nolint: makezero // appending is definitely correct

		cDesc[0] = "Failed in Consistently; " + optionalDescription[0].(string)
		cDesc = append(cDesc, optionalDescription[1:]...) //nolint: makezero // appending is definitely correct
	}

	gomega.Eventually(
		actualOrCtx, tk.EventuallyTimeout, tk.EventuallyPoll,
	).Should(matcher, eDesc...)

	return gomega.Consistently(
		actualOrCtx, tk.ConsistentlyTimeout, tk.ConsistentlyPoll,
	).Should(matcher, cDesc...)
}

// RegisterDebugMessage returns a pointer to a string which will be logged at the
// end of the test only if the test fails. This is particularly useful for logging
// information only once in an Eventually or Consistently function.
// Note: using a custom description message may be a better practice overall.
func RegisterDebugMessage() *string {
	var debugMsg string

	ginkgo.DeferCleanup(func() {
		if ginkgo.CurrentSpecReport().Failed() {
			ginkgo.GinkgoWriter.Println(debugMsg)
		}
	})

	return &debugMsg
}
