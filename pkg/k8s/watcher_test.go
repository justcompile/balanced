package k8s

import (
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
				watchNamespaces:   make(map[string]struct{}),
				excludeNamespaces: make(map[string]struct{}),
			},
			true,
		},
		"Should ignore if namespace has been excluded": {
			endpoint,
			&Watcher{
				watchNamespaces:   make(map[string]struct{}),
				excludeNamespaces: map[string]struct{}{"default": {}},
			},
			false,
		},
		"Should watch if namespace has not been excluded": {
			endpoint,
			&Watcher{
				watchNamespaces:   make(map[string]struct{}),
				excludeNamespaces: map[string]struct{}{"foobar": {}},
			},
			true,
		},
		"Should watch if namespace has been explicitly specified": {
			endpoint,
			&Watcher{
				watchNamespaces:   map[string]struct{}{"default": {}},
				excludeNamespaces: make(map[string]struct{}),
			},
			true,
		},
		"Should not watch if namespaces have been specified and isn't in that list": {
			endpoint,
			&Watcher{
				watchNamespaces:   map[string]struct{}{"foobar": {}},
				excludeNamespaces: make(map[string]struct{}),
			},
			false,
		},
	}

	for name, test := range tests {
		shouldWatch := shouldWatchResource(test.watcher, test.endpoint)

		assert.Equal(t, shouldWatch, test.shouldWatchObject, name)
	}
}
