package cloud

import (
	"balanced/pkg/configuration"
	"balanced/pkg/types"
)

const (
	SecurityGroupTag = "balanced:managed"
)

type CloudProvider interface {
	GetAddresses(*LookupConfig) ([]string, error)
	ReconcileSecurityGroups(map[string]*types.LoadBalancerUpstreamDefinition, bool) error
	UpsertRecordSet([]string) error
}

type SecurityGroup struct {
	Id    string
	Ports []int32
}

type LookupConfig struct {
	TagKey      string
	TagValue    string
	UsePublicIP bool
}

type initProvider func(*configuration.Config) (CloudProvider, error)

var registry = map[string]initProvider{}

func RegisterProvider(name string, f initProvider) {
	registry[name] = f
}

func GetProvider(name string, cfg *configuration.Config) (CloudProvider, error) {
	return registry[name](cfg)
}
