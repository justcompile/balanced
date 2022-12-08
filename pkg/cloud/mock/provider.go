package mock

import (
	"balanced/pkg/cloud"
	"balanced/pkg/configuration"
	"balanced/pkg/types"
)

type CloudProvider struct{}

func (c *CloudProvider) GetAddresses(*cloud.LookupConfig) ([]string, error) {
	return nil, nil
}
func (c *CloudProvider) ReconcileSecurityGroups(map[string]*types.LoadBalancerUpstreamDefinition, bool) error {
	return nil
}
func (c *CloudProvider) UpsertRecordSet([]string) error {
	return nil
}

func init() {
	cloud.RegisterProvider("mock", func(cfg *configuration.Config) (cloud.CloudProvider, error) {
		return &CloudProvider{}, nil
	})
}
