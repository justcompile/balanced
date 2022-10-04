package cmd

import (
	"balanced/pkg/k8s"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var ingress = &cobra.Command{
	Use: "ingress",
	Run: func(cmd *cobra.Command, args []string) {
		w, err := k8s.NewWatcher("/Users/richardhayes/.kube/argocd-eks-staging-us-east-1")
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

func init() {
	root.AddCommand(ingress)
}
