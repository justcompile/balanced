package k8s

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NamespacedResource interface {
	GetName() string
	GetNamespace() string
	GetOwnerReferences() []metav1.OwnerReference
}

func namespacedResourceToKey(ns NamespacedResource) *namespaceNameKey {
	name := ns.GetName()
	if len(ns.GetOwnerReferences()) > 0 {
		name = ns.GetOwnerReferences()[0].Name
	}

	return &namespaceNameKey{name: name, namespace: ns.GetNamespace()}
}

type namespaceNameKey struct {
	name      string
	namespace string
}

func (n *namespaceNameKey) String() string {
	return fmt.Sprintf("%s:%s", n.name, n.namespace)
}
