// Copyright Contributors to the Open Cluster Management project

package v1beta1

import (
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetCondition returns the existing index and condition on the status matching the given type. If
// no condition of that type is found, it will return -1 as the index.
func (status PolicyCoreStatus) GetCondition(condType string) (int, metav1.Condition) {
	for i, cond := range status.Conditions {
		if cond.Type == condType {
			return i, cond
		}
	}

	return -1, metav1.Condition{}
}

// UpdateCondition modifies the specified condition in the status or adds it if not present,
// ensuring conditions remain sorted by Type. Returns true if the condition was updated or added.
func (status *PolicyCoreStatus) UpdateCondition(newCond metav1.Condition) (changed bool) {
	idx, existingCond := status.GetCondition(newCond.Type)
	if idx == -1 {
		if newCond.LastTransitionTime.IsZero() {
			newCond.LastTransitionTime = metav1.Now()
		}

		status.Conditions = append(status.Conditions, newCond)

		sort.SliceStable(status.Conditions, func(i, j int) bool {
			return status.Conditions[i].Type < status.Conditions[j].Type
		})

		return true
	} else if condSemanticallyChanged(newCond, existingCond) {
		if newCond.LastTransitionTime.IsZero() {
			newCond.LastTransitionTime = metav1.Now()
		}

		status.Conditions[idx] = newCond

		// Do not sort in this case, assume that they are in order.

		return true
	}

	return false
}

func condSemanticallyChanged(newCond, oldCond metav1.Condition) bool {
	return newCond.Message != oldCond.Message ||
		newCond.Reason != oldCond.Reason ||
		newCond.Status != oldCond.Status
}
