package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:   "balanced",
	Short: "balanced .....",
}

func Execute() {
	if err := root.Execute(); err != nil {
		log.Fatal(err)
	}
}
