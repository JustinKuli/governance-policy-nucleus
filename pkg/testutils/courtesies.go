// Copyright Contributors to the Open Cluster Management project

package testutils

import (
	"regexp"
	"time"

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
func EventFilter(evs []corev1.Event, evType, msg string, since time.Time) []corev1.Event {
	msgRegex, err := regexp.Compile(msg)
	if err != nil {
		msgRegex = regexp.MustCompile(regexp.QuoteMeta(msg))
	}

	ans := make([]corev1.Event, 0)

	for _, ev := range evs {
		if evType != "" && ev.Type != evType {
			continue
		}

		if !msgRegex.MatchString(ev.Message) {
			continue
		}

		if !ev.EventTime.IsZero() && since.After(ev.EventTime.Time) {
			continue
		}

		ans = append(ans, ev)
	}

	return ans
}
