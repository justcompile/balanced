package configuration

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func patchEnvironment(valsToSet map[string]string) func() {
	current := make(map[string]string)
	for k, v := range valsToSet {
		current[k] = os.Getenv(k)
		os.Setenv(k, v)
	}

	return func() {
		for k, v := range current {
			os.Setenv(k, v)
		}
	}
}

func TestBalancedConfig_GetConfigPath(t *testing.T) {
	tests := map[string]struct {
		cfg          *KubeConfig
		envVars      map[string]string
		expectedPath string
	}{
		"returns kubeconfig set on config obj if set": {
			&KubeConfig{ConfigPath: "my/kube/config"},
			nil,
			"my/kube/config",
		},
		"returns value from KUBECONFIG env var if set": {
			&KubeConfig{},
			map[string]string{"KUBECONFIG": "env/var/kube/config"},
			"env/var/kube/config",
		},
		"returns value from homedir": {
			&KubeConfig{},
			map[string]string{"KUBECONFIG": "", "HOME": "/foobar"},
			"/foobar/.kube/config",
		},
	}

	for name, test := range tests {
		defer patchEnvironment(test.envVars)()

		path := test.cfg.GetConfigPath()
		assert.Equal(t, test.expectedPath, path, name)
	}
}

func createTempFile(data string) (*os.File, error) {
	f, err := os.CreateTemp("", "balanced*")
	if err != nil {
		return nil, err
	}

	if data != "" {
		if _, wErr := f.WriteString(data); wErr != nil {
			return nil, wErr
		}
	}
	return f, nil
}

func TestConfig_New(t *testing.T) {
	tests := map[string]struct {
		seed        func() (string, error)
		expectedErr error
		expectedCfg *Config
	}{
		"returns error when file does not exist": {
			func() (string, error) { return "does not exist", nil },
			errors.New("configuration: open does not exist: no such file or directory"),
			nil,
		},
		"returns config object with kube config path set": {
			func() (string, error) {
				data := "[kubernetes]\nkube-config = \"/foobar/kube/config\""
				f, err := createTempFile(data)
				if err != nil {
					return "", err
				}

				defer f.Close()
				return f.Name(), nil
			},
			nil,
			&Config{Kubernetes: &KubeConfig{ConfigPath: "/foobar/kube/config"}},
		},
		"returns config object when custom dns commands are provided": {
			func() (string, error) {
				data := "[dns]\nenabled = true\n\n[dns.custom]\nadd-command = \"dns.sh add something\""
				f, err := createTempFile(data)
				if err != nil {
					return "", err
				}

				defer f.Close()
				return f.Name(), nil
			},
			nil,
			&Config{DNS: DNS{Enabled: true, Custom: &CustomDNS{AddCommand: "dns.sh add something"}}},
		},
	}

	for name, test := range tests {
		filePath, fErr := test.seed()
		if fErr != nil {
			t.Fatal(fErr, name)
		}

		defer os.Remove(filePath)

		cfg, err := New(filePath)
		// assert.Equal(t, filePath, "", name)
		assert.Equal(t, test.expectedErr, err, name)
		assert.Equal(t, test.expectedCfg, cfg, name)
	}
}
