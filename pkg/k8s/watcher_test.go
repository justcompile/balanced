package k8s

import (
	"balanced/pkg/types"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	k8stypes "k8s.io/apimachinery/pkg/types"
	k8sTesting "k8s.io/client-go/testing"
)

func TestNamespaceFiltering(t *testing.T) {
	endpoint := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		},
	}

	tests := map[string]struct {
		endpoint          *discoveryv1.EndpointSlice
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

func Test_getEndpointFromService(t *testing.T) {
	tests := map[string]struct {
		input            *corev1.Service
		seed             func(*fake.Clientset)
		expectedError    error
		expectedEndpoint *discoveryv1.EndpointSlice
	}{
		"returns error when no endpoint slice can be found": {
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					UID:       k8stypes.UID("id-1234"),
				},
			},
			func(cs *fake.Clientset) {},
			fmt.Errorf("could not locate EndpointSlice for service foo:bar"),
			nil,
		},
		"returns error when error calling endpointslice.list occurs": {
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					UID:       k8stypes.UID("id-1234"),
				},
			},
			func(cs *fake.Clientset) {
				cs.PrependReactor("list", "endpointslices", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, fmt.Errorf("unable to list endpointslices")
				})
			},
			fmt.Errorf("unable to list endpointslices"),
			nil,
		},
		"returns error no endpointslices are owned by the requesting service": {
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					UID:       k8stypes.UID("id-1234"),
				},
			},
			func(cs *fake.Clientset) {
				cs.DiscoveryV1().EndpointSlices("foo").Create(t.Context(), &discoveryv1.EndpointSlice{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dave-1234",
						Namespace: "bar",
						OwnerReferences: []metav1.OwnerReference{
							{
								UID: k8stypes.UID("nope"),
							},
						},
					},
				}, metav1.CreateOptions{})
			},
			fmt.Errorf("could not locate EndpointSlice for service foo:bar"),
			nil,
		},
		"returns endpoint slice owned by service": {
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					UID:       k8stypes.UID("id-1234"),
				},
			},
			func(cs *fake.Clientset) {
				cs.PrependReactor("list", "endpointslices", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, &discoveryv1.EndpointSliceList{
						Items: []discoveryv1.EndpointSlice{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "dave-1234",
									Namespace: "bar",
									OwnerReferences: []metav1.OwnerReference{
										{
											UID: k8stypes.UID("id-1234"),
										},
									},
								},
							},
						},
					}, nil
				})
			},
			nil,
			&discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dave-1234",
					Namespace: "bar",
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: k8stypes.UID("id-1234"),
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		clientSet := fake.NewClientset()

		w := &Watcher{clientset: clientSet}

		test.seed(clientSet)

		endpointSlice, err := w.getEndpointFromService(test.input)

		assert.Equal(t, test.expectedError, err, name)
		assert.Equal(t, test.expectedEndpoint, endpointSlice, name)
	}
}
