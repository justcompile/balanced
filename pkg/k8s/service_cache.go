package k8s

import (
	"balanced/pkg/configuration"
	"context"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type serviceCache struct {
	cfg           *configuration.KubeConfig
	clientset     kubernetes.Interface
	domainMapping map[string]string
	mx            *sync.RWMutex
}

func (s *serviceCache) lookupDomainForService(ctx context.Context, ns *namespaceNameKey) string {
	s.mx.RLock()
	defer s.mx.RUnlock()
	if _, exists := s.domainMapping[ns.String()]; !exists {
		domain, err := s.getDomainFromServiceAnnotation(ctx, ns)
		if err != nil {
			log.Error(err.Error())
			return ""
		}

		if domain != "" {
			s.domainMapping[ns.String()] = domain
		}
	}

	return s.domainMapping[ns.String()]
}

func (s *serviceCache) getDomainFromServiceAnnotation(ctx context.Context, ns *namespaceNameKey) (string, error) {
	svc, err := s.clientset.CoreV1().Services(ns.namespace).Get(ctx, ns.name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error retrieving service %s => %s", ns, err.Error())
	}

	var domain string
	var exists bool

	if domain, exists = svc.GetAnnotations()[s.cfg.ServiceAnnotationKey]; !exists {
		return "", fmt.Errorf("annotation %s cannot be found on service %s", s.cfg.ServiceAnnotationKey, ns)
	}

	return domain, nil
}

func (s *serviceCache) removeServiceRecord(ctx context.Context, ns *namespaceNameKey) {
	s.mx.Lock()
	defer s.mx.Unlock()

	delete(s.domainMapping, ns.String())
}

func newServiceCache(cfg *configuration.KubeConfig, clientset kubernetes.Interface) *serviceCache {
	return &serviceCache{
		cfg:           cfg,
		clientset:     clientset,
		domainMapping: make(map[string]string),
		mx:            &sync.RWMutex{},
	}
}
