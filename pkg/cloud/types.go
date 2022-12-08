package cloud

import (
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
