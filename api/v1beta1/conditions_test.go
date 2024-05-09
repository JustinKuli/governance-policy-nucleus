// Copyright Contributors to the Open Cluster Management project

package v1beta1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getSampleStatus() PolicyCoreStatus {
	return PolicyCoreStatus{
		Conditions: []metav1.Condition{{
			Type:    "Apple",
			Status:  metav1.ConditionTrue,
			Reason:  "NoDoctor",
			Message: "an apple a day...",
		}, {
			Type:    "Compliant",
			Status:  metav1.ConditionTrue,
			Reason:  "Compliant",
			Message: "everything is good",
		}, {
			Type:    "Bonus",
			Status:  metav1.ConditionFalse,
			Reason:  "Compliant",
			Message: "this condition is out of order",
		}},
	}
}

func TestGetCondition(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		inputType string
		wantIdx   int
		wantMsg   string
	}{
		"Apple is found": {
			inputType: "Apple",
			wantIdx:   0,
			wantMsg:   "an apple a day...",
		},
		"Compliant is found": {
			inputType: "Compliant",
			wantIdx:   1,
			wantMsg:   "everything is good",
		},
		"Imaginary is not found": {
			inputType: "Imaginary",
			wantIdx:   -1,
			wantMsg:   "",
		},
	}

	for name, tcase := range tests {
		gotIdx, gotCond := getSampleStatus().GetCondition(tcase.inputType)

		if gotIdx != tcase.wantIdx {
			t.Errorf("Expected index %v in test %q, got %v", tcase.wantIdx, name, gotIdx)
		}

		if tcase.wantIdx == -1 {
			continue
		}

		if gotCond.Message != tcase.wantMsg {
			t.Errorf("Expected message %q in test %q, got %q", tcase.wantMsg, name, gotCond.Message)
		}
	}
}

func TestUpdateCondition(t *testing.T) {
	t.Parallel()

	baseLen := len(getSampleStatus().Conditions)

	tests := map[string]struct {
		newCond    metav1.Condition
		wantChange bool
		wantLen    int
		wantIdx    int
	}{
		"Imaginary should be added to the end": {
			newCond: metav1.Condition{
				Type:    "Imaginary",
				Status:  metav1.ConditionFalse,
				Reason:  "Existent",
				Message: "not just imaginary",
			},
			wantChange: true,
			wantLen:    baseLen + 1,
			wantIdx:    baseLen,
		},
		"Basic should be added in the middle": {
			newCond: metav1.Condition{
				Type:    "Basic",
				Status:  metav1.ConditionTrue,
				Reason:  "Easy",
				Message: "should be simple enough",
			},
			wantChange: true,
			wantLen:    baseLen + 1,
			wantIdx:    1,
		},
		"Bonus should be updated but not moved": {
			newCond: metav1.Condition{
				Type:    "Bonus",
				Status:  metav1.ConditionTrue,
				Reason:  "Compliant",
				Message: "this condition is now in order",
			},
			wantChange: true,
			wantLen:    baseLen,
			wantIdx:    baseLen - 1,
		},
		"Apple should be updated if the message is different": {
			newCond: metav1.Condition{
				Type:    "Apple",
				Status:  metav1.ConditionTrue,
				Reason:  "NoDoctor",
				Message: "an apple a day keeps the doctor away",
			},
			wantChange: true,
			wantLen:    baseLen,
			wantIdx:    0,
		},
		"Apple should not be updated if the message is the same": {
			newCond: metav1.Condition{
				Type:    "Apple",
				Status:  metav1.ConditionTrue,
				Reason:  "NoDoctor",
				Message: "an apple a day...",
			},
			wantChange: false,
			wantLen:    baseLen,
			wantIdx:    0,
		},
	}

	for name, tcase := range tests {
		status := getSampleStatus()

		gotChanged := status.UpdateCondition(tcase.newCond)

		if tcase.wantChange && !gotChanged {
			t.Errorf("Expected test %q to change the conditions, but it didn't", name)
		} else if !tcase.wantChange && gotChanged {
			t.Errorf("Expected test %q not to change the conditions, but it did", name)
		}

		gotLen := len(status.Conditions)
		if gotLen != tcase.wantLen {
			t.Errorf("Expected conditions to be length %v after test %q, got length %v",
				tcase.wantLen, name, gotLen)
		}

		gotIdx, gotCond := status.GetCondition(tcase.newCond.Type)

		if gotIdx != tcase.wantIdx {
			t.Errorf("Expected condition to be at index %v after test %q, got index %v",
				tcase.wantIdx, name, gotIdx)
		}

		if gotCond.Message != tcase.newCond.Message {
			t.Errorf("Expected condition to have message %q after test %q, got %q",
				tcase.newCond.Message, name, gotCond.Message)
		}
	}
}
