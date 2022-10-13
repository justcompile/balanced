package k8s

import (
	"context"
	"errors"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockClientset struct {
	services []*v1.Service
	kubernetes.Interface
}

func (cs *mockClientset) CoreV1() typedcorev1.CoreV1Interface {
	return &mockCoreV1{services: cs.services}
}

type mockCoreV1 struct {
	services []*v1.Service
	typedcorev1.CoreV1Interface
}

func (c *mockCoreV1) Services(string) typedcorev1.ServiceInterface {
	return &mockServices{services: c.services}
}

type mockServices struct {
	services []*v1.Service
	typedcorev1.ServiceInterface
}

func (s *mockServices) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Service, error) {
	for _, svc := range s.services {
		if svc.Name == name {
			return svc, nil
		}
	}

	return nil, errors.New("does not exist")
}
