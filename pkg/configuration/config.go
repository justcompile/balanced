package configuration

import (
	"fmt"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"os"
	"path/filepath"

	"k8s.io/client-go/util/homedir"
	"k8s.io/utils/ptr"
)

var (
	defaultSyncInterval = time.Second * 20
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
	HostedZoneId string `toml:"route-53-hosted-zone-id"`
	Type         string `toml:"route-53-record-type"`
	TTL          int64  `toml:"route-53-ttl"`
}

type CustomDNS struct {
	AddCommand    string `toml:"add-command"`
	RemoveCommand string `toml:"remove-command"`
}

type DNS struct {
	Enabled          bool   `toml:"enabled"`
	Address          string `toml:"advertised-address"`
	UsePublicAddress bool   `toml:"use-public-address"`

	Custom *CustomDNS `toml:"custom"`
}

type LoadBalancer struct {
	ConfigDir         string                  `toml:"config-dir"`
	DataPlane         *HAProxyDataPlaneConfig `toml:"data-plane-api"`
	ReconcileDuration *time.Duration          `toml:"sync-interval"`
	ReloadCmd         string                  `toml:"reload-cmd"`
	Template          string                  `toml:"template"`
}

func (lb LoadBalancer) UseDataPlaneAPI() bool {
	if lb.DataPlane == nil {
		return false
	}

	// default to true if the block has been defined but not explicitly enabled/disabled
	if lb.DataPlane.Enabled == nil {
		return true
	}

	return ptr.Deref(lb.DataPlane.Enabled, false)
}

type HAProxyDataPlaneConfig struct {
	Enabled  *bool  `toml:"enabled"`
	Endpoint string `toml:"endpoint"`
	Version  string `toml:"api-version"`
	Username string `toml:"username"`
	Password string `toml:"password"`
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

func (k *KubeConfig) HealthCheckAnnotationKey() string {
	prefix := strings.TrimSuffix(k.ServiceAnnotationKeyPrefix, "/")
	return fmt.Sprintf("%s/health-check", prefix)
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

	if cfg.LoadBalancer != nil && cfg.LoadBalancer.ReconcileDuration == nil {
		cfg.LoadBalancer.ReconcileDuration = &defaultSyncInterval
	}
	return &cfg, nil
}
