package configuration

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"

	"os"
	"path/filepath"

	"k8s.io/client-go/util/homedir"
)

type Config struct {
	Kubernetes   *KubeConfig
	LoadBalancer *LoadBalancer
	Cloud        Cloud
	DNS          DNS
}

type Cloud struct {
	AWS *AWS
}

type AWS struct {
	HostedZoneId string   `toml:"route-53-hosted-zone-id"`
	Type         string   `toml:"route-53-record-type"`
	TTL          int64    `toml:"route-53-ttl"`
	Tags         []string `toml:"tags"`
}

func (a *AWS) TagsAsMap() map[string]string {
	val := make(map[string]string)

	for _, tag := range a.Tags {
		parts := strings.Split(tag, "=")
		val[parts[0]] = parts[1]
	}

	return val
}

type DNS struct {
	Enabled          bool   `toml:"enabled"`
	TagKey           string `toml:"discovery-tag"`
	UsePublicAddress bool   `toml:"use-public-address"`
}

type LoadBalancer struct {
	ConfigDir string `toml:"config-dir"`
	ReloadCmd string `toml:"reload-cmd"`
	Template  string `toml:"template"`
}

type KubeConfig struct {
	ConfigPath                      string   `toml:"kube-config"`
	ServiceAnnotationKeyPrefix      string   `toml:"service-annotation-key-prefix"`
	ServiceAnnotationLoadBalancerId string   `toml:"service-annotation-load-balancer-id"`
	WatchedNamespaces               []string `toml:"watch-namespaces"`
	ExcludedNamespaces              []string `toml:"exclude-namespaces"`
}

func (k *KubeConfig) DomainAnnotationKey() string {
	prefix := strings.TrimSuffix(k.ServiceAnnotationKeyPrefix, "/")
	return fmt.Sprintf("%s/domains", prefix)
}

func (k *KubeConfig) LoadBalancerIdAnnotationKey() string {
	prefix := strings.TrimSuffix(k.ServiceAnnotationKeyPrefix, "/")
	return fmt.Sprintf("%s/load-balancer-id", prefix)
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
		return nil, fmt.Errorf("configuration: %s", err)
	}
	return &cfg, nil
}
