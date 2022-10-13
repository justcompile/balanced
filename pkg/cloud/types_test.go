package cloud

import (
	"balanced/pkg/configuration"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderFromConfig(t *testing.T) {
	tests := map[string]struct {
		input           *configuration.DNS
		expectedProvder CloudProvider
		expectedErr     error
	}{
		"returns error when DNS provider is not configured": {
			&configuration.DNS{},
			nil,
			errors.New("no dns provider has been defined"),
		},
	}

	for name, test := range tests {
		p, err := ProviderFromConfig(test.input)
		assert.Equal(t, test.expectedErr, err, name)
		assert.Equal(t, test.expectedProvder, p, name)
	}
}
