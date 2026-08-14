package cmd

import (
	"balanced/pkg/configuration"
	"balanced/pkg/dataplane"
	"balanced/pkg/types"
	"encoding/json"
	"log"

	"github.com/spf13/cobra"
)

var tmp = &cobra.Command{
	Use: "test",
	Run: func(cmd *cobra.Command, args []string) {
		c, err := dataplane.NewDataPlaneAPIClient(configuration.HAProxyDataPlaneConfig{
			Endpoint: "http://localhost:5555",
			Username: "admin",
			Password: "lr6jdlqL",
		})
		if err != nil {
			log.Fatal(err)
		}

		def := &types.LoadBalancerUpstreamDefinition{Domain: args[0], HealthCheck: "/health"}

		b, err := c.GetOrCreateBackend(def)
		if err != nil {
			log.Fatal(err)
		}

		enc := json.NewEncoder(log.Writer())
		enc.SetIndent("", "  ")

		enc.Encode(b)

		s, err := c.GetServers(b)
		if err != nil {
			log.Fatal(err)
		}

		enc.Encode(s)
	},
}

func init() {
	root.AddCommand(tmp)
}
