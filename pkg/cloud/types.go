package cloud

import (
	"balanced/pkg/configuration"
	"fmt"
)

type LookupConfig struct {
	TagKey      string
	TagValue    string
	UsePublicIP bool
}

type CloudProvider interface {
	GetAddresses(*LookupConfig) ([]string, error)
	UpsertRecordSet([]string) error
}

func ProviderFromConfig(d *configuration.DNS) (CloudProvider, error) {
	if d.Route53 != nil {
		return NewAWSProvider(d)
	}

	return nil, fmt.Errorf("no dns provider has been defined")
}
