package controller

import (
	"balanced/pkg/configuration"
	"balanced/pkg/dataplane"
	"balanced/pkg/dns"
	"balanced/pkg/fileupdater"
	"balanced/pkg/types"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"
)

const (
	retryAttempts = 3
)

type LoadBalancerUpdater interface {
	IsReloadRequired() bool
	ReloadProcess() error
	Update(change *types.LoadBalancerUpstreamDefinition) error
}

type Controller struct {
	cfg     *configuration.Config
	dns     dns.Registrar
	updater LoadBalancerUpdater
}

func (c *Controller) OnExit() error {
	if c.dns != nil {
		return c.dns.RemoveAll()
	}

	return nil
}

func (c *Controller) Start(changes chan *types.Change) {
	ticker := time.NewTicker(*c.cfg.LoadBalancer.ReconcileDuration)
	defer ticker.Stop()

	for {
		select {
		case change, ok := <-changes:
			if !ok {
				return
			}

			if !c.shouldChange(change) {
				changes <- change
				continue
			}

			if err := c.updater.Update(change.Obj); err != nil {
				log.Error(err)
				change.Retried += 1
				if change.Retried < retryAttempts {
					log.Infof("retry %d/%d: reschedule change for %s", change.Retried, retryAttempts, change.Obj.Domain)
					change.RetryAfter = ptr.To(time.Now().Add(time.Second * 5))
					changes <- change
				} else {
					log.Infof("retry %d/%d: change for %s could not be applied", change.Retried, retryAttempts, change.Obj.Domain)
				}
				continue
			}

			if err := c.dns.Add(change.Obj.Domain); err != nil {
				log.Errorf("unable to update DNS record for %s: %s", change.Obj.Domain, err)
			}
		case <-ticker.C:
			if c.updater.IsReloadRequired() {
				if reloadErr := c.updater.ReloadProcess(); reloadErr != nil {
					log.Error(reloadErr)
				}

				log.Debugf("process reloaded successfully")
			}
		}
	}
}

func (u *Controller) shouldChange(change *types.Change) bool {
	if change.RetryAfter != nil {
		if time.Now().Before(*change.RetryAfter) {
			return false
		}
	}

	return true
}

func NewController(cfg *configuration.Config) (*Controller, error) {
	reg, err := dns.NewDNSRegistrar(&cfg.DNS)
	if err != nil {
		return nil, err
	}

	var lbu LoadBalancerUpdater
	var lbuErr error

	if cfg.LoadBalancer.UseDataPlaneAPI() {
		lbu, lbuErr = dataplane.NewUpdater(cfg)
		log.Info("using dataplane updater")
	} else {
		lbu, lbuErr = fileupdater.NewUpdater(cfg)
		log.Info("using config file updater")
	}

	if lbuErr != nil {
		return nil, err
	}

	return &Controller{
		cfg:     cfg,
		dns:     reg,
		updater: lbu,
	}, nil
}
