package dns

import (
	"balanced/pkg/configuration"
	"fmt"
)

type Updater interface {
	UpsertRecordSet([]string) error
}

func UpdaterFromConfig(d *configuration.DNS) (Updater, error) {
	if d.Route53 != nil {
		return NewRoute53Updater(d)
	}

	return nil, fmt.Errorf("no dns provider has been defined")
}
