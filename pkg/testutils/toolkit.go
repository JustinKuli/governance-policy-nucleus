// Copyright Contributors to the Open Cluster Management project

package testutils

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	gomegaTypes "github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/api/errors"
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
func (tk Toolkit) CleanlyCreate(ctx context.Context, obj client.Object) error {
	// Save and then re-set the GVK because the API call removes it
	savedGVK := obj.GetObjectKind().GroupVersionKind()
	createErr := tk.Create(ctx, obj)
	obj.GetObjectKind().SetGroupVersionKind(savedGVK)

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

// EC runs assertions on asynchronous behavior, both *E*ventually and *C*onsistently,
// using the polling and timeout settings of the toolkit. Its usage should feel familiar
// to gomega users, simply skip the `.Should(...)` call and put your matcher as the second
// parameter here.
func (tk Toolkit) EC(
	actualOrCtx interface{}, matcher gomegaTypes.GomegaMatcher, optionalDescription ...interface{},
) bool {
	gomega.Eventually(
		actualOrCtx, tk.EventuallyTimeout, tk.EventuallyPoll,
	).Should(matcher, optionalDescription...)

	return gomega.Consistently(
		actualOrCtx, tk.ConsistentlyTimeout, tk.ConsistentlyPoll,
	).Should(matcher, optionalDescription...)
}
