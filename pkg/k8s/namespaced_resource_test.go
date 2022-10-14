package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_namespacedResourceToKey(t *testing.T) {
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
	}

	r := namespacedResourceToKey(p)

	assert.Equal(t, "foo", r.name)
	assert.Equal(t, "bar", r.namespace)
}

func Test_namespaceNameKey_String(t *testing.T) {
	r := &namespaceNameKey{name: "foo", namespace: "fizz"}

	assert.Equal(t, "foo:fizz", r.String())
}
