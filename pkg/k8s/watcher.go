package k8s

import (
	"balanced/pkg/configuration"
	"balanced/pkg/types"
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
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
		excludeNamespaces: make(map[string]struct{}),
		watchNamespaces:   make(map[string]struct{}),
		serviceCache:      newServiceCache(cfg, clientset),
	}

	for _, ns := range cfg.WatchedNamespaces {
		w.watchNamespaces[ns] = struct{}{}
	}

	for _, ns := range cfg.ExcludedNamespaces {
		w.excludeNamespaces[ns] = struct{}{}
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
	watchNamespaces   map[string]struct{}
	excludeNamespaces map[string]struct{}
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
				w.serviceCache.removeServiceRecord(context.Background(), namespacedResourceToKey(svc))
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

			key := namespacedResourceToKey(endpoint)

			domain := w.serviceCache.lookupDomainForService(context.Background(), key)
			if domain == "" {
				return
			}

			log.Infof("endpoint added: %s", key)

			c <- types.NewLoadBalancerDefinitionChange(domain, endpoint)
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
				key := namespacedResourceToKey(newEndpoint)

				log.Infof("endpoint %s changed", key)

				domain := w.serviceCache.lookupDomainForService(context.Background(), key)
				if domain == "" {
					return
				}
				c <- types.NewLoadBalancerDefinitionChange(domain, newEndpoint)
			}
		},
	})

	w.informer = kubeInformerFactory
	return c
}
