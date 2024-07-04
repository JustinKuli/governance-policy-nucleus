// Copyright Contributors to the Open Cluster Management project

package testutils

import (
	"regexp"
	"time"

	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ObjNN returns a NamespacedName for the given Object.
func ObjNN(obj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

// EventFilter filters the given events. Any of the filter parameters can be passed an empty
// value to ignore that field when filtering. The msg parameter will be compiled into a regex if
// possible. The since parameter checks against the event's EventTime - but if the event does not
// specify an EventTime, it will not be filtered out.
func EventFilter(events []corev1.Event, evType, msg string, since time.Time) []corev1.Event {
	msgRegex, err := regexp.Compile(msg)
	if err != nil {
		msgRegex = regexp.MustCompile(regexp.QuoteMeta(msg))
	}

	ans := make([]corev1.Event, 0)

	for i := range events {
		if evType != "" && events[i].Type != evType {
			continue
		}

		if !msgRegex.MatchString(events[i].Message) {
			continue
		}

		if !events[i].EventTime.IsZero() && since.After(events[i].EventTime.Time) {
			continue
		}

		ans = append(ans, events[i])
	}

	return ans
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
