package k8s

import (
	"balanced/pkg/configuration"
	"context"
	"errors"
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
		"returns error if annotation not found on service": {
			[]*v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "foo",
						Namespace:   "bar",
						Annotations: make(map[string]string),
					},
				},
			},
			&namespaceNameKey{name: "foo", namespace: "bar"},
			"",
			errors.New("annotation my.uri/domain cannot be found on service foo:bar"),
		},
		"returns domain if annotation found on service": {
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
			"foobar.com",
			nil,
		},
	}

	for name, test := range tests {
		s := &serviceCache{
			cfg: &configuration.KubeConfig{
				ServiceAnnotationKey: "my.uri/domain",
			},
			clientset: &mockClientset{services: test.services},
		}

		domain, err := s.getDomainFromServiceAnnotation(context.TODO(), test.namespaceKey)

		assert.Equal(t, test.expectedErr, err, name)
		assert.Equal(t, test.expectedDomain, domain, name)
	}
}
