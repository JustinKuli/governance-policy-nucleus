// Copyright Contributors to the Open Cluster Management project

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// fakeObjList minimally implements client.ObjectList. It is only to be used for tests.
type fakeObjList string

// ensure fakeObjList implements client.ObjectList.
var _ client.ObjectList = (*fakeObjList)(nil)

func (l fakeObjList) GetResourceVersion() string {
	return string(l)
}

func (l fakeObjList) SetResourceVersion(_ string) {
}

func (l fakeObjList) GetSelfLink() string {
	return string(l)
}

func (l fakeObjList) SetSelfLink(_ string) {
}

func (l fakeObjList) GetContinue() string {
	return string(l)
}

func (l fakeObjList) SetContinue(_ string) {
}

func (l fakeObjList) GetRemainingItemCount() *int64 {
	return nil
}

func (l fakeObjList) SetRemainingItemCount(_ *int64) {
}

func (l fakeObjList) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}

func (l fakeObjList) DeepCopyObject() runtime.Object {
	return l
}

// fakeObjList minimally implements client.Object. It is only to be used for tests.
type fakeObj string

// ensure fakeObj implements client.Object.
var _ client.Object = (*fakeObj)(nil)

func (o fakeObj) GetNamespace() string {
	return string(o)
}

func (o fakeObj) SetNamespace(_ string) {
}

func (o fakeObj) GetName() string {
	return string(o)
}

func (o fakeObj) SetName(_ string) {
}

func (o fakeObj) GetGenerateName() string {
	return string(o)
}

func (o fakeObj) SetGenerateName(_ string) {
}

func (o fakeObj) GetUID() types.UID {
	return types.UID(o)
}

func (o fakeObj) SetUID(_ types.UID) {
}

func (o fakeObj) GetResourceVersion() string {
	return string(o)
}

func (o fakeObj) SetResourceVersion(_ string) {
}

func (o fakeObj) GetGeneration() int64 {
	return 0
}

func (o fakeObj) SetGeneration(_ int64) {
}

func (o fakeObj) GetSelfLink() string {
	return string(o)
}

func (o fakeObj) SetSelfLink(_ string) {
}

func (o fakeObj) GetCreationTimestamp() metav1.Time {
	return metav1.Now()
}

func (o fakeObj) SetCreationTimestamp(_ metav1.Time) {
}

func (o fakeObj) GetDeletionTimestamp() *metav1.Time {
	return nil
}

func (o fakeObj) SetDeletionTimestamp(_ *metav1.Time) {
}

func (o fakeObj) GetDeletionGracePeriodSeconds() *int64 {
	return nil
}

func (o fakeObj) SetDeletionGracePeriodSeconds(*int64) {
}

func (o fakeObj) GetLabels() map[string]string {
	return nil
}

func (o fakeObj) SetLabels(_ map[string]string) {
}

func (o fakeObj) GetAnnotations() map[string]string {
	return nil
}

func (o fakeObj) SetAnnotations(_ map[string]string) {
}

func (o fakeObj) GetFinalizers() []string {
	return nil
}

func (o fakeObj) SetFinalizers(_ []string) {
}

func (o fakeObj) GetOwnerReferences() []metav1.OwnerReference {
	return nil
}

func (o fakeObj) SetOwnerReferences([]metav1.OwnerReference) {
}

func (o fakeObj) GetManagedFields() []metav1.ManagedFieldsEntry {
	return nil
}

func (o fakeObj) SetManagedFields(_ []metav1.ManagedFieldsEntry) {
}

func (o fakeObj) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}

func (o fakeObj) DeepCopyObject() runtime.Object {
	return o
}
