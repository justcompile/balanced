package k8s

import (
	"balanced/pkg/configuration"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type serviceCache struct {
	cfg           *configuration.KubeConfig
	clientset     kubernetes.Interface
	domainMapping map[string]*serviceData
	mx            *sync.RWMutex
}

type serviceData struct {
	domains             []string
	healthCheckEndpoint string
}

func (s *serviceCache) lookupService(ctx context.Context, ns *namespaceNameKey) *serviceData {
	s.mx.RLock()
	defer s.mx.RUnlock()
	if _, exists := s.domainMapping[ns.String()]; !exists {
		svc, err := s.getService(ctx, ns)
		if err != nil {
			log.Error(err.Error())
			return nil
		}

		domains, err := s.getDomainFromServiceAnnotation(svc, ns)
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
			d := &serviceData{
				domains:             domains,
				healthCheckEndpoint: s.tryGetHealthCheckEndpointFromServiceAnnotation(svc, ns),
			}
			s.domainMapping[ns.String()] = d
		}
	}

	return s.domainMapping[ns.String()]
}

func (s *serviceCache) getService(ctx context.Context, ns *namespaceNameKey) (*corev1.Service, error) {
	svc, err := s.clientset.CoreV1().Services(ns.namespace).Get(ctx, ns.name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error retrieving service %s => %s", ns, err.Error())
	}

	return svc, nil
}

func (s *serviceCache) getDomainFromServiceAnnotation(svc *corev1.Service, ns *namespaceNameKey) ([]string, error) {

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

func (s *serviceCache) tryGetHealthCheckEndpointFromServiceAnnotation(svc *corev1.Service, ns *namespaceNameKey) string {
	var endpoint string
	var exists bool

	endpoint, exists = svc.GetAnnotations()[s.cfg.HealthCheckAnnotationKey()]

	if !exists {
		log.Debugf("service %s does not have health check annotation set, using default", ns)
		return "/health"
	}

	return endpoint
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
		domainMapping: make(map[string]*serviceData),
		mx:            &sync.RWMutex{},
	}
}
