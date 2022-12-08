package k8s

import (
	"balanced/pkg/configuration"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type serviceCache struct {
	cfg           *configuration.KubeConfig
	clientset     kubernetes.Interface
	domainMapping map[string][]string
	mx            *sync.RWMutex
}

func (s *serviceCache) lookupDomainForService(ctx context.Context, ns *namespaceNameKey) []string {
	s.mx.RLock()
	defer s.mx.RUnlock()
	if _, exists := s.domainMapping[ns.String()]; !exists {
		domains, err := s.getDomainFromServiceAnnotation(ctx, ns)
		if err != nil {
			var ign *IgnoreService
			if errors.As(err, &ign) {
				log.Warn(err)
			} else {
				log.Errorf("%T", err)
				log.Error(err.Error())
			}
			return nil
		}

		if len(domains) > 0 {
			s.domainMapping[ns.String()] = domains
		}
	}

	return s.domainMapping[ns.String()]
}

func (s *serviceCache) getDomainFromServiceAnnotation(ctx context.Context, ns *namespaceNameKey) ([]string, error) {
	svc, err := s.clientset.CoreV1().Services(ns.namespace).Get(ctx, ns.name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error retrieving service %s => %s", ns, err.Error())
	}

	var domain string
	var exists bool

	annotations := svc.GetAnnotations()
	if id := annotations[s.cfg.LoadBalancerIdAnnotationKey()]; id != s.cfg.ServiceAnnotationLoadBalancerId {
		return nil, &IgnoreService{service: ns.String(), reason: fmt.Sprintf("annotation %s empty or does not match this load balancer id: %s", s.cfg.LoadBalancerIdAnnotationKey(), s.cfg.ServiceAnnotationLoadBalancerId)}
	}

	if domain, exists = annotations[s.cfg.DomainAnnotationKey()]; !exists {
		return nil, &IgnoreService{service: ns.String(), reason: fmt.Sprintf("annotation %s cannot be found", s.cfg.DomainAnnotationKey())}
	}

	return strings.Split(domain, ","), nil
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
		domainMapping: make(map[string][]string),
		mx:            &sync.RWMutex{},
	}
}
