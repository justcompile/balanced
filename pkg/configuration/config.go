package configuration

import (
	"github.com/BurntSushi/toml"

	"os"
	"path/filepath"

	"k8s.io/client-go/util/homedir"
)

type Config struct {
	Kubernetes   *KubeConfig
	LoadBalancer *LoadBalancer
}

type LoadBalancer struct {
	ConfigDir string `toml:"config-dir"`
	ReloadCmd string `toml:"reload-cmd"`
	Template  string `toml:"template"`
}

type KubeConfig struct {
	ConfigPath           string   `toml:"kube-config"`
	ServiceAnnotationKey string   `toml:"service-annotation-key"`
	WatchedNamespaces    []string `toml:"watch-namespaces"`
	ExcludedNamespaces   []string `toml:"exclude-namespaces"`
}

func (k *KubeConfig) GetConfigPath() string {
	if k.ConfigPath != "" {
		return k.ConfigPath
	}

	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}

	home := homedir.HomeDir()
	return filepath.Join(home, ".kube", "config")
}

func New(path string) (*Config, error) {
	var cfg Config
	_, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
