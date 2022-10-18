package k8s

import (
	"balanced/pkg/types"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNamespaceFiltering(t *testing.T) {
	endpoint := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		},
	}

	tests := map[string]struct {
		endpoint          *corev1.Endpoints
		watcher           *Watcher
		shouldWatchObject bool
	}{
		"Should watch if no filters have been applied": {
			endpoint,
			&Watcher{
				watchNamespaces:   make(types.Set[string]),
				excludeNamespaces: make(types.Set[string]),
			},
			true,
		},
		"Should ignore if namespace has been excluded": {
			endpoint,
			&Watcher{
				watchNamespaces:   make(types.Set[string]),
				excludeNamespaces: types.Set[string]{"default": {}},
			},
			false,
		},
		"Should watch if namespace has not been excluded": {
			endpoint,
			&Watcher{
				watchNamespaces:   make(types.Set[string]),
				excludeNamespaces: types.Set[string]{"foobar": {}},
			},
			true,
		},
		"Should watch if namespace has been explicitly specified": {
			endpoint,
			&Watcher{
				watchNamespaces:   types.Set[string]{"default": {}},
				excludeNamespaces: make(types.Set[string]),
			},
			true,
		},
		"Should not watch if namespaces have been specified and isn't in that list": {
			endpoint,
			&Watcher{
				watchNamespaces:   types.Set[string]{"foobar": {}},
				excludeNamespaces: make(types.Set[string]),
			},
			false,
		},
	}

	for name, test := range tests {
		shouldWatch := shouldWatchResource(test.watcher, test.endpoint)

		assert.Equal(t, shouldWatch, test.shouldWatchObject, name)
	}
}
