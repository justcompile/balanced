package k8s

import (
	"balanced/pkg/types"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func sliceToSetMap(val []string) types.Set[string] {
	s := make(types.Set[string])

	for _, v := range val {
		s.Add(v)
	}

	return s
}

func Test_endpointHasChanged(t *testing.T) {
	tests := map[string]struct {
		oldEndpoint    *corev1.Endpoints
		newEndpoint    *corev1.Endpoints
		expectedResult bool
	}{
		"has not changed if resource versions match": {
			&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "a",
				},
			},
			&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "a",
				},
			},
			false,
		},
		"has not changed if ip addresses match but unordered": {
			&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "a",
				},
				Subsets: []corev1.EndpointSubset{
					{Addresses: []corev1.EndpointAddress{
						{IP: "10.1.1.2"},
					}},
					{Addresses: []corev1.EndpointAddress{
						{IP: "10.1.1.3"},
					}},
					{Addresses: []corev1.EndpointAddress{
						{IP: "10.1.1.1"},
					}},
				},
			},
			&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "b",
				},
				Subsets: []corev1.EndpointSubset{
					{Addresses: []corev1.EndpointAddress{
						{IP: "10.1.1.3"},
					}},
					{Addresses: []corev1.EndpointAddress{
						{IP: "10.1.1.2"},
					}},
					{Addresses: []corev1.EndpointAddress{
						{IP: "10.1.1.1"},
					}},
				},
			},
			false,
		},
		"has not changed if ip addresses match": {
			&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "a",
				},
				Subsets: []corev1.EndpointSubset{
					{Addresses: []corev1.EndpointAddress{
						{IP: "10.1.1.1"},
					}},
				},
			},
			&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "b",
				},
				Subsets: []corev1.EndpointSubset{
					{Addresses: []corev1.EndpointAddress{
						{IP: "10.1.1.1"},
					}},
				},
			},
			false,
		},
		"has changed if ip addresses do not match": {
			&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "a",
				},
				Subsets: []corev1.EndpointSubset{
					{Addresses: []corev1.EndpointAddress{
						{IP: "10.1.1.1"},
					}},
				},
			},
			&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "b",
				},
				Subsets: []corev1.EndpointSubset{
					{Addresses: []corev1.EndpointAddress{
						{IP: "10.1.1.10"},
					}},
				},
			},
			true,
		},
		"has changed if number of ip addresses do not match": {
			&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "a",
				},
				Subsets: []corev1.EndpointSubset{
					{Addresses: []corev1.EndpointAddress{
						{IP: "10.1.1.2"},
					}},
					{Addresses: []corev1.EndpointAddress{
						{IP: "10.1.1.1"},
					}},
				},
			},
			&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "b",
				},
				Subsets: []corev1.EndpointSubset{
					{Addresses: []corev1.EndpointAddress{
						{IP: "10.1.1.1"},
					}},
				},
			},
			true,
		},
	}

	for name, test := range tests {
		res := endpointHasChanged(test.oldEndpoint, test.newEndpoint)
		assert.Equal(t, test.expectedResult, res, name)
	}
}

func Test_shouldWatchResource(t *testing.T) {
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
	}

	tests := map[string]struct {
		excludedNamespaces []string
		watchNamespaces    []string
		expectedResult     bool
	}{
		"should watch if not excluded and watched namespaces are empty": {
			nil,
			nil,
			true,
		},
		"should watch if explicitly specified": {
			nil,
			[]string{"bar"},
			true,
		},
		"should not watch if excluded": {
			[]string{"bar"},
			nil,
			false,
		},
		"should not watch if not explicitly specified": {
			nil,
			[]string{"boo"},
			false,
		},
	}

	for name, test := range tests {
		w := &Watcher{
			excludeNamespaces: sliceToSetMap(test.excludedNamespaces),
			watchNamespaces:   sliceToSetMap(test.watchNamespaces),
		}

		res := shouldWatchResource(w, p)
		assert.Equal(t, test.expectedResult, res, name)
	}
}
