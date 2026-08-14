package cmd

import (
	"balanced/pkg/configuration"
	"balanced/pkg/controller"
	"balanced/pkg/k8s"
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
		cfgPath, _ := cmd.Flags().GetString("config")
		cfg, err := configuration.New(cfgPath)
		if err != nil {
			log.Fatal(err)
		}

		ctrl, err := controller.NewController(cfg)

		if err != nil {
			log.Fatal(err)
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
		go ctrl.Start(changes)

		select {
		case <-cmd.Context().Done():
		case <-sig:
			close(changes)
			stop <- struct{}{}
			log.Println("stopping")
			if err := ctrl.OnExit(); err != nil {
				log.Errorln(err.Error())
			}
			time.Sleep(time.Second)
		}
	},
}

func Execute() {
	root.PersistentFlags().StringP("config", "c", "./balanced.toml", "Path to config file")

	if err := root.Execute(); err != nil {
		log.Fatal(err)
	}
}
