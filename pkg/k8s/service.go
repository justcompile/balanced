package k8s

import (
	"fmt"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
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

func kubeconfigPath(path string) string {
	if path != "" {
		return path
	}

	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}

	home := homedir.HomeDir()
	return filepath.Join(home, ".kube", "config")
}

func NewWatcher(configPath string, opts ...WatchOptions) (*Watcher, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath(configPath))
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
	ingressInformer := kubeInformerFactory.Core().V1().Endpoints().Informer()

	ingressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			endpoint := obj.(*corev1.Endpoints)
			if endpoint.Namespace == "default" {
				fmt.Printf("endpoint added: %s \n", endpoint.Name)
			}
		},
		DeleteFunc: func(obj interface{}) {
			fmt.Printf("endpoint deleted: %s \n", obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldEndpoint := oldObj.(*corev1.Endpoints)
			newEndpoint := newObj.(*corev1.Endpoints)
			if oldEndpoint.Namespace == "default" {
				if oldEndpoint.GetResourceVersion() != newEndpoint.GetResourceVersion() {
					f := log.Fields{
						"old": w.ipsFromEndpoint(oldEndpoint),
						"new": w.ipsFromEndpoint(newEndpoint),
					}
					log.WithFields(f).Infof("endpoint changed: %s", newEndpoint.Name)
				}
			}
		},
	})

	w.informer = kubeInformerFactory
	w.isSetup = true
}

func (w *Watcher) ipsFromEndpoint(e *corev1.Endpoints) []string {
	addresses := make([]string, 0)
	for _, ss := range e.Subsets {
		for _, a := range ss.Addresses {
			addresses = append(addresses, a.IP)
		}
	}

	return addresses
}