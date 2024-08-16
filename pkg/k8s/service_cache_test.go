package k8s

import (
	"balanced/pkg/configuration"
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func TestServiceCache_getDomainFromServiceAnnotation(t *testing.T) {
	tests := map[string]struct {
		service        *v1.Service
		namespaceKey   *namespaceNameKey
		expectedResult []string
		expectedErr    error
	}{
		"returns error if domain annotation not found on service": {
			&v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						"my.uri/load-balancer-id": "testing",
					},
				},
			},
			&namespaceNameKey{name: "foo", namespace: "bar"},
			nil,
			&IgnoreService{service: "foo:bar", reason: "annotation my.uri/domains cannot be found"},
		},
		"returns domain if annotation found on service and lb id matches": {
			&v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						"my.uri/domains":          "foobar.com",
						"my.uri/load-balancer-id": "testing",
					},
				},
			},
			&namespaceNameKey{name: "foo", namespace: "bar"},
			[]string{"foobar.com"},
			nil,
		},
		"returns ignore error if annotation found on service but id does not match": {
			&v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						"my.uri/domains": "foobar.com",
					},
				},
			},
			&namespaceNameKey{name: "foo", namespace: "bar"},
			nil,
			&IgnoreService{service: "foo:bar", reason: "annotation my.uri/load-balancer-id empty or does not match this load balancer id: testing"},
		},
	}

	for name, test := range tests {
		s := &serviceCache{
			cfg: &configuration.KubeConfig{
				ServiceAnnotationKeyPrefix:      "my.uri",
				ServiceAnnotationLoadBalancerId: "testing",
			},
			clientset: &mockClientset{services: []*v1.Service{test.service}},
		}

		domains, err := s.getDomainFromServiceAnnotation(test.service, test.namespaceKey)

		assert.Equal(t, test.expectedErr, err, name)
		assert.Equal(t, test.expectedResult, domains, name)
	}
}

func TestServiceCache_getService(t *testing.T) {
	tests := map[string]struct {
		clientset       kubernetes.Interface
		namespaceKey    *namespaceNameKey
		expectedService *v1.Service
		expectedErr     error
	}{
		"returns error if service cannot be found": {
			&mockClientset{},
			&namespaceNameKey{name: "foo", namespace: "bar"},
			nil,
			errors.New("error retrieving service foo:bar => does not exist"),
		},
		"returns service if found": {
			&mockClientset{services: []*v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						Annotations: map[string]string{
							"my.uri/load-balancer-id": "testing",
						},
					},
				}},
			},
			&namespaceNameKey{name: "foo", namespace: "bar"},
			&v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						"my.uri/load-balancer-id": "testing",
					},
				},
			},
			nil,
		},
	}

	for name, test := range tests {
		s := &serviceCache{
			cfg: &configuration.KubeConfig{
				ServiceAnnotationKeyPrefix:      "my.uri",
				ServiceAnnotationLoadBalancerId: "testing",
			},
			clientset: test.clientset,
		}

		svc, err := s.getService(context.Background(), test.namespaceKey)

		assert.Equal(t, test.expectedErr, err, name)
		assert.Equal(t, test.expectedService, svc, name)
	}
}

func TestServiceCache_lookupService(t *testing.T) {
	tests := map[string]struct {
		services       []*v1.Service
		cache          map[string]*serviceData
		namespaceKey   *namespaceNameKey
		expectedResult *serviceData
		expectedErr    error
	}{
		"returns error if service is not in cache and cannot be found": {
			nil,
			make(map[string]*serviceData),
			&namespaceNameKey{name: "foo", namespace: "bar"},
			nil,
			errors.New("error retrieving service foo:bar => does not exist"),
		},
		"returns domain from cache if already set": {
			nil, // will result in an error if value is not in cache
			map[string]*serviceData{"foo:bar": {domains: []string{"foobar.example.com"}}},
			&namespaceNameKey{name: "foo", namespace: "bar"},
			&serviceData{domains: []string{"foobar.example.com"}},
			nil,
		},
		"does not retrieve domain from service annotation if available but id does not match": {
			[]*v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						Annotations: map[string]string{
							"my.uri/domains": "foobar.com",
						},
					},
				},
			},
			make(map[string]*serviceData),
			&namespaceNameKey{name: "foo", namespace: "bar"},
			nil,
			nil,
		},
		"retrieves domain from service annotation if available and id matches": {
			[]*v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						Annotations: map[string]string{
							"my.uri/domains":          "foobar.com",
							"my.uri/load-balancer-id": "testing",
						},
					},
				},
			},
			make(map[string]*serviceData),
			&namespaceNameKey{name: "foo", namespace: "bar"},
			&serviceData{domains: []string{"foobar.com"}, healthCheckEndpoint: "/health"},
			nil,
		},
	}

	for name, test := range tests {
		s := newServiceCache(
			&configuration.KubeConfig{
				ServiceAnnotationKeyPrefix:      "my.uri",
				ServiceAnnotationLoadBalancerId: "testing",
			},
			&mockClientset{services: test.services},
		)

		s.domainMapping = test.cache

		domains := s.lookupService(context.TODO(), test.namespaceKey)

		assert.Equal(t, test.expectedResult, domains, name)
	}
}

func TestServiceCache_removeServiceRecord(t *testing.T) {
	tests := map[string]struct {
		initialCache map[string]*serviceData
		namespaceKey *namespaceNameKey
		expected     map[string]*serviceData
	}{
		"does not panic if record does not exist": {
			make(map[string]*serviceData),
			&namespaceNameKey{name: "foo", namespace: "bar"},
			make(map[string]*serviceData),
		},
		"does remove record if record exists": {
			map[string]*serviceData{"foo:bar": {}},
			&namespaceNameKey{name: "foo", namespace: "bar"},
			make(map[string]*serviceData),
		},
		"cache is unaffected if record does not exist": {
			map[string]*serviceData{"foo:bar": {}},
			&namespaceNameKey{name: "fizz", namespace: "bar"},
			map[string]*serviceData{"foo:bar": {}},
		},
	}

	for name, test := range tests {
		s := &serviceCache{
			domainMapping: test.initialCache,
			mx:            &sync.RWMutex{},
		}

		s.removeServiceRecord(context.TODO(), test.namespaceKey)

		assert.Equal(t, test.expected, s.domainMapping, name)
	}
}
