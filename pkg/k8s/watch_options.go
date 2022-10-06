package k8s

import "time"

type WatchOptions func(*Watcher)

func WithResyncInterval(i time.Duration) WatchOptions {
	return func(w *Watcher) {
		w.resyncInterval = &i
	}
}
