package k8s

import "fmt"

type NamespacedResource interface {
	GetName() string
	GetNamespace() string
}

func namespacedResourceToKey(ns NamespacedResource) *namespaceNameKey {
	return &namespaceNameKey{name: ns.GetName(), namespace: ns.GetNamespace()}
}

type namespaceNameKey struct {
	name      string
	namespace string
}

func (n *namespaceNameKey) String() string {
	return fmt.Sprintf("%s:%s", n.name, n.namespace)
}
