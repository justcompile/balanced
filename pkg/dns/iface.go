package dns

import (
	"balanced/pkg/configuration"
	"fmt"
)

type lookupConfig struct {
	usePublicIP bool
	tagKey      string
	tagValue    string
}

type Updater interface {
	GetAddresses() ([]string, error)
	UpsertRecordSet([]string) error
}

func UpdaterFromConfig(d *configuration.DNS) (Updater, error) {
	if d.Route53 != nil {
		return NewRoute53Updater(d)
	}

	return nil, fmt.Errorf("no dns provider has been defined")
}
