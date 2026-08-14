package dataplane

import (
	"balanced/pkg/configuration"
	"balanced/pkg/types"
	"errors"
)

type Updater struct {
	cfg    configuration.HAProxyDataPlaneConfig
	client *DataPlaneAPIClient
}

func (u *Updater) IsReloadRequired() bool {
	return false
}

func (u *Updater) ReloadProcess() error {
	// implemented to conform to interface
	return nil
}

func (u *Updater) Update(change *types.LoadBalancerUpstreamDefinition) error {
	_, err := u.client.GetOrCreateBackend(change)
	if err != nil {
		return err
	}

	return u.client.UpdateServers(change)
}

func (u *Updater) setupClient() (err error) {
	u.client, err = NewDataPlaneAPIClient(u.cfg)
	return err
}

func NewUpdater(cfg *configuration.Config) (*Updater, error) {
	if !cfg.LoadBalancer.UseDataPlaneAPI() {
		return nil, errors.New("configuration has not been setup to interact with the Data Plane API")
	}

	u := &Updater{cfg: *cfg.LoadBalancer.DataPlane}

	return u, u.setupClient()
}
