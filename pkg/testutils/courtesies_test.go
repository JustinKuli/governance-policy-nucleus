// Copyright Contributors to the Open Cluster Management project

package testutils

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestObjNN(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		inpObj   client.Object
		wantName string
		wantNS   string
	}{
		"namespaced unstructured": {
			inpObj: &unstructured.Unstructured{Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "foo",
					"namespace": "world",
				},
			}},
			wantName: "foo",
			wantNS:   "world",
		},
		"cluster-scoped unstructured": {
			inpObj: &unstructured.Unstructured{Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "bar",
				},
			}},
			wantName: "bar",
			wantNS:   "",
		},
		"(namespaced) configmap": {
			inpObj: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Name:      "my-cm",
				Namespace: "kube-one",
			}},
			wantName: "my-cm",
			wantNS:   "kube-one",
		},
		"(cluster-scoped) node": {
			inpObj: &corev1.Node{ObjectMeta: metav1.ObjectMeta{
				Name: "unit-tests-only",
			}},
			wantName: "unit-tests-only",
			wantNS:   "",
		},
	}

	for name, tcase := range tests {
		got := ObjNN(tcase.inpObj)

		if got.Name != tcase.wantName {
			t.Errorf("Wanted name '%v', got '%v' in test '%v'", tcase.wantName, got.Name, name)
		}

		if got.Namespace != tcase.wantNS {
			t.Errorf("Wanted namespace '%v', got '%v' in test '%v'", tcase.wantNS, got.Namespace, name)
		}
	}
}

func TestEventFilter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	old := now.Add(-time.Minute)
	veryOld := now.Add(-time.Hour)

	sampleEvents := []corev1.Event{{
		Message:   "hello",
		Type:      "Normal",
		EventTime: metav1.NewMicroTime(veryOld),
	}, {
		Message:   "goodbye",
		Type:      "Warning",
		EventTime: metav1.NewMicroTime(old),
	}, {
		Message:   "carpe diem [",
		Type:      "Normal",
		EventTime: metav1.NewMicroTime(now),
	}, {
		Message: "what time is it?",
		Type:    "Warning",
	}}

	tests := map[string]struct {
		inpType  string
		inpMsg   string
		inpSince time.Time
		wantIdxs []int
	}{
		"#NoFilter": {
			inpType:  "",
			inpMsg:   "",
			inpSince: time.Time{},
			wantIdxs: []int{0, 1, 2, 3},
		},
		"recent events, plus the one with no time specified": {
			inpType:  "",
			inpMsg:   "",
			inpSince: now.Add(-5 * time.Minute),
			wantIdxs: []int{1, 2, 3},
		},
		"only warnings": {
			inpType:  "Warning",
			inpMsg:   "",
			inpSince: time.Time{},
			wantIdxs: []int{1, 3},
		},
		"basic regex for a space": {
			inpType:  "",
			inpMsg:   ".* .*",
			inpSince: time.Time{},
			wantIdxs: []int{2, 3},
		},
		"just a space": {
			inpType:  "",
			inpMsg:   " ",
			inpSince: time.Time{},
			wantIdxs: []int{2, 3},
		},
		"invalid inescaped regex": {
			inpType:  "",
			inpMsg:   "[",
			inpSince: time.Time{},
			wantIdxs: []int{2},
		},
	}

	for name, tcase := range tests {
		got := EventFilter(sampleEvents, tcase.inpType, tcase.inpMsg, tcase.inpSince)

		if len(got) != len(tcase.wantIdxs) {
			t.Fatalf("Expected %v events to be returned, got %v in test %v: got events: %v",
				len(tcase.wantIdxs), len(got), name, got)
		}

		for i, wantIdx := range tcase.wantIdxs {
			if sampleEvents[wantIdx].String() != got[i].String() {
				t.Errorf("Mismatch on item #%v in test %v. Expected '%v' got '%v'",
					i, name, sampleEvents[wantIdx], got[i])
			}
		}
	}
}
