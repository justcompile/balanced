package k8s

import (
	"balanced/pkg/configuration"
	"balanced/pkg/types"
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func NewWatcher(cfg *configuration.KubeConfig, opts ...WatchOptions) (*Watcher, error) {
	config, err := clientcmd.BuildConfigFromFlags("", cfg.GetConfigPath())
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		cfg:               cfg,
		clientset:         clientset,
		excludeNamespaces: make(types.Set[string]),
		watchNamespaces:   make(types.Set[string]),
		serviceCache:      newServiceCache(cfg, clientset),
	}

	for _, ns := range cfg.WatchedNamespaces {
		w.watchNamespaces.Add(ns)
	}

	for _, ns := range cfg.ExcludedNamespaces {
		w.excludeNamespaces.Add(ns)
	}

	for _, opt := range opts {
		opt(w)
	}

	if w.resyncInterval == nil {
		defaultInterval := time.Second * 30
		w.resyncInterval = &defaultInterval
	}

	return w, nil
}

type Watcher struct {
	cfg               *configuration.KubeConfig
	clientset         *kubernetes.Clientset
	resyncInterval    *time.Duration
	informer          kubeinformers.SharedInformerFactory
	watchNamespaces   types.Set[string]
	excludeNamespaces types.Set[string]
	serviceCache      *serviceCache
}

func (w *Watcher) Start(stop chan struct{}) chan *types.Change {
	c := w.setup()
	w.informer.Start(stop)

	return c
}

func (w *Watcher) setup() chan *types.Change {
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(w.clientset, *w.resyncInterval)
	endpointsInformer := kubeInformerFactory.Core().V1().Endpoints().Informer()
	serviceInformer := kubeInformerFactory.Core().V1().Services().Informer()

	c := make(chan *types.Change)

	// when a service is updated, this would mean that an annotation may have been added/updated
	// clear the domain mapping cache to ensure that it can be picked up
	serviceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			svc := oldObj.(*corev1.Service)
			if shouldWatchResource(w, svc) {
				key := namespacedResourceToKey(svc)
				w.serviceCache.removeServiceRecord(context.Background(), key)

				endpoint, err := w.getEndpointFromService(svc)
				if err != nil {
					log.Errorf("unable to retrieve endpoint for svc: %s", key)
					return
				}

				w.handleChange(c, endpoint)
			}
		},
		DeleteFunc: func(obj interface{}) {
			svc := obj.(*corev1.Service)
			if shouldWatchResource(w, svc) {
				w.serviceCache.removeServiceRecord(context.Background(), namespacedResourceToKey(svc))
			}
		},
	})

	endpointsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			endpoint := obj.(*corev1.Endpoints)
			if !shouldWatchResource(w, endpoint) {
				log.Debugf("endpoint added but namespace %s is not being watched", endpoint.GetNamespace())
				return
			}

			w.handleChange(c, endpoint)
		},
		DeleteFunc: func(obj interface{}) {
			key := namespacedResourceToKey(obj.(*corev1.Endpoints))
			log.Infof("endpoint deleted: %s", key)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldEndpoint := oldObj.(*corev1.Endpoints)
			newEndpoint := newObj.(*corev1.Endpoints)

			if !shouldWatchResource(w, oldEndpoint) {
				log.Debugf("endpoint changed but namespace %s is not being watched", oldEndpoint.GetNamespace())
				return
			}

			if endpointHasChanged(oldEndpoint, newEndpoint) {
				w.handleChange(c, newEndpoint)
			}
		},
	})

	w.informer = kubeInformerFactory
	return c
}

func (w *Watcher) getEndpointFromService(s *corev1.Service) (*corev1.Endpoints, error) {
	return w.clientset.CoreV1().Endpoints(s.Namespace).Get(context.Background(), s.Name, metav1.GetOptions{})
}

func (w *Watcher) handleChange(c chan *types.Change, e *corev1.Endpoints) {
	key := namespacedResourceToKey(e)

	svc := w.serviceCache.lookupService(context.Background(), key)

	if svc == nil || len(svc.domains) == 0 {
		return
	}
	for _, domain := range svc.domains {
		def := types.NewLoadBalancerDefinitionChange(domain, svc.healthCheckEndpoint, e)

		if len(def.Obj.Servers) == 0 {
			log.Warnf("endpoint %s changed but endpoint has 0 ready addresses", key)
			continue
		}

		log.Infof("endpoint %s changed, queuing update", key)

		c <- def
	}
}
