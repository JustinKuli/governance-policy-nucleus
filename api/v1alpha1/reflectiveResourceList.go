package v1alpha1

import (
	"fmt"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"open-cluster-management.io/governance-policy-nucleus/api/v1beta1"
)

//+kubebuilder:object:generate=false

// ReflectiveResourceList implements ResourceList for the wrapped client.ObjectList, by using
// reflection. The wrapped list must have an Items field, with a slice of items which satisfy the
// client.Object interface - most types which satisfy client.ObjectList seem to follow this
// convention. Using this type is not recommended: implementing ResourceList yourself will generally
// lead to better performance.
type ReflectiveResourceList struct {
	ClientList client.ObjectList
}

// ensure ReflectiveResourceList implements ResourceList
var _ v1beta1.ResourceList = (*ReflectiveResourceList)(nil)

// Items returns the list of items in the list. Since this implementation uses reflection, it may
// have errors or not perform as well as a bespoke implementation for the underlying type. The
// returned Objects are in the same order that they are in the list.
func (l *ReflectiveResourceList) Items() ([]client.Object, error) {
	value := reflect.ValueOf(l.ClientList)
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return nil, &ReflectiveResourceListError{
			typeName: value.Type().PkgPath() + "." + value.Type().Name(),
			message:  "the underlying go Kind was not a struct",
		}
	}

	itemsField := value.FieldByName("Items")
	if !itemsField.IsValid() {
		return nil, &ReflectiveResourceListError{
			typeName: value.Type().PkgPath() + "." + value.Type().Name(),
			message:  "the underlying struct does not have a field called 'Items'",
		}
	}

	if itemsField.Kind() != reflect.Slice {
		return nil, &ReflectiveResourceListError{
			typeName: value.Type().PkgPath() + "." + value.Type().Name(),
			message:  "the 'Items' field in the underlying struct isn't a slice",
		}
	}

	items := make([]client.Object, itemsField.Len())

	for i := 0; i < itemsField.Len(); i++ {
		item, ok := itemsField.Index(i).Interface().(client.Object)
		if ok {
			items[i] = item

			continue
		}

		// Try a pointer receiver
		item, ok = itemsField.Index(i).Addr().Interface().(client.Object)
		if ok {
			items[i] = item

			continue
		}

		return nil, &ReflectiveResourceListError{
			typeName: value.Type().PkgPath() + "." + value.Type().Name(),
			message: "an item in the underlying struct's 'Items' slice could not be " +
				"type-asserted to a sigs.k8s.io/controller-runtime/pkg/client.Object",
		}
	}

	return items, nil
}

func (l *ReflectiveResourceList) ObjectList() client.ObjectList {
	return l.ClientList
}

//+kubebuilder:object:generate=false

type ReflectiveResourceListError struct {
	typeName string
	message  string
}

func (e *ReflectiveResourceListError) Error() string {
	return fmt.Sprintf("unable to use %v as a nucleus ResourceList: %v", e.typeName, e.message)
}
