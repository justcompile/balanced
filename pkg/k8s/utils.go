package k8s

import (
	"balanced/pkg/types"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func shouldWatchResource[T NamespacedResource](w *Watcher, obj T) bool {
	return (w.watchNamespaces.Has(obj.GetNamespace()) || len(w.watchNamespaces) == 0) && !w.excludeNamespaces.Has(obj.GetNamespace())
}

func endpointHasChanged(oldEndpoint, newEndpoint *corev1.Endpoints) bool {
	if oldEndpoint.GetResourceVersion() != newEndpoint.GetResourceVersion() {
		oldIps := types.SortedIPsFromEndpoint(oldEndpoint)
		newIps := types.SortedIPsFromEndpoint(newEndpoint)

		return !equal(oldIps, newIps)
	}
	return false
}

func equal[T fmt.Stringer](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v.String() != b[i].String() {
			return false
		}
	}

	return true
}
