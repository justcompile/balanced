package cmd

import (
	"balanced/pkg/configuration"
	"balanced/pkg/k8s"
	"balanced/pkg/loadbalancer"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:   "balanced",
	Short: "balanced .....",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := configuration.New()
		if err != nil {
			log.Fatal(err)
		}

		lb, lbErr := loadbalancer.NewUpdater(cfg.LoadBalancer)
		if lbErr != nil {
			log.Fatal(lbErr)
		}

		w, err := k8s.NewWatcher(cfg.Kubernetes)
		if err != nil {
			log.Fatal(err)
		}

		sig := make(chan os.Signal, 1)

		signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

		stop := make(chan struct{})
		defer close(stop)

		// Start watching for Endpoint Changes
		changes := w.Start(stop)

		// Start update process listening to changes which come in
		go lb.Start(changes)

		select {
		case <-cmd.Context().Done():
		case <-sig:
			close(changes)
			stop <- struct{}{}
			log.Println("stopping")
			time.Sleep(time.Second)
		}
	},
}

func Execute() {
	if err := root.Execute(); err != nil {
		log.Fatal(err)
	}
}
