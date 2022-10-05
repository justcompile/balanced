package cmd

import (
	"balanced/pkg/configuration"
	"balanced/pkg/k8s"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

var root = &cobra.Command{
	Use:   "balanced",
	Short: "balanced .....",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := configuration.New()
		if err != nil {
			log.Fatal(err)
		}

		w, err := k8s.NewWatcher(cfg.Kubernetes.ConfigPath)
		if err != nil {
			log.Fatal(err)
		}

		sig := make(chan os.Signal, 1)

		signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

		stop := make(chan struct{})
		defer close(stop)

		w.Start(stop)
		log.Println("watching...")

		select {
		case <-cmd.Context().Done():
		case <-sig:
			stop <- struct{}{}
			log.Println("stopping")
		}
	},
}

func Execute() {
	if err := root.Execute(); err != nil {
		log.Fatal(err)
	}
}
