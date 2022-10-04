package k8s

import (
	"fmt"
	"time"

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

func NewWatcher(configPath string, opts ...WatchOptions) (*Watcher, error) {
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		clientset: clientset,
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

type WatchOptions func(*Watcher)

func WithResyncInterval(i time.Duration) WatchOptions {
	return func(w *Watcher) {
		w.resyncInterval = &i
	}
}

type Watcher struct {
	clientset      *kubernetes.Clientset
	resyncInterval *time.Duration
	informer       kubeinformers.SharedInformerFactory
	isSetup        bool
}

func (w *Watcher) Start(stop chan struct{}) {
	if !w.isSetup {
		w.setup()
	}

	w.informer.Start(stop)
}

func (w *Watcher) setup() {
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(w.clientset, *w.resyncInterval)
	ingressInformer := kubeInformerFactory.Networking().V1().Ingresses().Informer()

	ingressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			fmt.Printf("ingress added: %s \n", obj)
		},
		DeleteFunc: func(obj interface{}) {
			fmt.Printf("ingress deleted: %s \n", obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			fmt.Printf("ingress changed: %s \n", newObj)
		},
	})

	w.informer = kubeInformerFactory
	w.isSetup = true
}
