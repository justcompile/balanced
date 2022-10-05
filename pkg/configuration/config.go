package configuration

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	Kubernetes *kubernetes
}

type kubernetes struct {
	ConfigPath string `toml:"kube-config"`
}

func New() (*Config, error) {
	var cfg Config
	_, err := toml.DecodeFile("balanced.toml", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
