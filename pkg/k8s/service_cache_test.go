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
)

func TestServiceCache_getDomainFromServiceAnnotation(t *testing.T) {
	tests := map[string]struct {
		services       []*v1.Service
		namespaceKey   *namespaceNameKey
		expectedDomain string
		expectedErr    error
	}{
		"returns error if service cannot be found": {
			nil,
			&namespaceNameKey{name: "foo", namespace: "bar"},
			"",
			errors.New("error retrieving service foo:bar => does not exist"),
		},
		"returns error if domain annotation not found on service": {
			[]*v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						Annotations: map[string]string{
							"my.uri/load-balancer-id": "testing",
						},
					},
				},
			},
			&namespaceNameKey{name: "foo", namespace: "bar"},
			"",
			&ignoreService{service: "foo:bar", reason: "annotation my.uri/domain cannot be found"},
		},
		"returns domain if annotation found on service and lb id matches": {
			[]*v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						Annotations: map[string]string{
							"my.uri/domain":           "foobar.com",
							"my.uri/load-balancer-id": "testing",
						},
					},
				},
			},
			&namespaceNameKey{name: "foo", namespace: "bar"},
			"foobar.com",
			nil,
		},
		"returns ignore error if annotation found on service but id does not match": {
			[]*v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						Annotations: map[string]string{
							"my.uri/domain": "foobar.com",
						},
					},
				},
			},
			&namespaceNameKey{name: "foo", namespace: "bar"},
			"",
			&ignoreService{service: "foo:bar", reason: "annotation my.uri/load-balancer-id empty or does not match this load balancer id: testing"},
		},
	}

	for name, test := range tests {
		s := &serviceCache{
			cfg: &configuration.KubeConfig{
				ServiceAnnotationKeyPrefix:      "my.uri",
				ServiceAnnotationLoadBalancerId: "testing",
			},
			clientset: &mockClientset{services: test.services},
		}

		domain, err := s.getDomainFromServiceAnnotation(context.TODO(), test.namespaceKey)

		assert.Equal(t, test.expectedErr, err, name)
		assert.Equal(t, test.expectedDomain, domain, name)
	}
}

func TestServiceCache_lookupDomainForService(t *testing.T) {
	tests := map[string]struct {
		services       []*v1.Service
		cache          map[string]string
		namespaceKey   *namespaceNameKey
		expectedDomain string
		expectedErr    error
	}{
		"returns error if service is not in cache and cannot be found": {
			nil,
			make(map[string]string),
			&namespaceNameKey{name: "foo", namespace: "bar"},
			"",
			errors.New("error retrieving service foo:bar => does not exist"),
		},
		"returns domain from cache if already set": {
			nil, // will result in an error if value is not in cache
			map[string]string{"foo:bar": "foobar.example.com"},
			&namespaceNameKey{name: "foo", namespace: "bar"},
			"foobar.example.com",
			nil,
		},
		"does not retrieve domain from service annotation if available but id does not match": {
			[]*v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						Annotations: map[string]string{
							"my.uri/domain": "foobar.com",
						},
					},
				},
			},
			make(map[string]string),
			&namespaceNameKey{name: "foo", namespace: "bar"},
			"",
			nil,
		},
		"retrieves domain from service annotation if available and id matches": {
			[]*v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						Annotations: map[string]string{
							"my.uri/domain":           "foobar.com",
							"my.uri/load-balancer-id": "testing",
						},
					},
				},
			},
			make(map[string]string),
			&namespaceNameKey{name: "foo", namespace: "bar"},
			"foobar.com",
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

		domain := s.lookupDomainForService(context.TODO(), test.namespaceKey)

		assert.Equal(t, test.expectedDomain, domain, name)
	}
}

func TestServiceCache_removeServiceRecord(t *testing.T) {
	tests := map[string]struct {
		initialCache map[string]string
		namespaceKey *namespaceNameKey
		expected     map[string]string
	}{
		"does not panic if record does not exist": {
			make(map[string]string),
			&namespaceNameKey{name: "foo", namespace: "bar"},
			make(map[string]string),
		},
		"does remove record if record exists": {
			map[string]string{"foo:bar": "fizzbuzz"},
			&namespaceNameKey{name: "foo", namespace: "bar"},
			make(map[string]string),
		},
		"cache is unaffected if record does not exist": {
			map[string]string{"foo:bar": "fizzbuzz"},
			&namespaceNameKey{name: "fizz", namespace: "bar"},
			map[string]string{"foo:bar": "fizzbuzz"},
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
