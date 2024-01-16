package v1alpha1

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestReflectiveItemsErrors(t *testing.T) {
	t.Parallel()

	type wrappedFakeObjList struct {
		fakeObjList
	}

	type wrappedWithIntItems struct {
		fakeObjList
		Items int
	}

	type wrappedWithBadItems struct {
		fakeObjList
		Items []int
	}

	type wrappedWithGoodItems struct {
		fakeObjList
		Items []fakeObj
	}

	tests := map[string]struct {
		list client.ObjectList
		msg  string
	}{
		"NamespaceList": {
			list: &corev1.NamespaceList{Items: []corev1.Namespace{{}}},
			msg:  "",
		},
		"fakeObjList": {
			list: fakeObjList(""),
			msg:  "the underlying go Kind was not a struct",
		},
		"wrappedFakeObjList": {
			list: wrappedFakeObjList{fakeObjList("")},
			msg:  "the underlying struct does not have a field called 'Items'",
		},
		"wrappedWithIntItems": {
			list: wrappedWithIntItems{fakeObjList: fakeObjList("")},
			msg:  "the 'Items' field in the underlying struct isn't a slice",
		},
		"wrappedWithEmptyItems": {
			list: wrappedWithBadItems{fakeObjList: fakeObjList(""), Items: []int{}},
			msg:  "",
		},
		"wrappedWithBadItems": {
			list: wrappedWithBadItems{fakeObjList: fakeObjList(""), Items: []int{0}},
			msg: "an item in the underlying struct's 'Items' slice could not be type-asserted " +
				"to a sigs.k8s.io/controller-runtime/pkg/client.Object",
		},
		"wrappedWithGoodItems": {
			list: wrappedWithGoodItems{fakeObjList: fakeObjList(""), Items: []fakeObj{fakeObj("")}},
			msg:  "",
		},
	}

	for name, tcase := range tests {
		refList := ReflectiveResourceList{ClientList: tcase.list}

		_, err := refList.Items()

		if tcase.msg == "" {
			if err != nil {
				t.Errorf("Unexpected error in test '%v', expected nil, got %v", name, err)
			}
		} else {
			if err == nil {
				t.Errorf("Expected an error in test '%v', but got nil", name)
			}

			wantErr := fmt.Sprintf("unable to use open-cluster-management.io/governance-policy-nucleus/api/v1alpha1."+
				"%v as a nucleus ResourceList: %v", name, tcase.msg)

			diff := cmp.Diff(wantErr, err.Error())
			if diff != "" {
				t.Errorf("Error mismatch in test '%v', diff: '%v'", name, diff)
			}
		}
	}
}
