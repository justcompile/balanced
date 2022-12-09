package loadbalancer

import (
	"balanced/pkg/cloud"
	"balanced/pkg/cloud/awscloud"
	"balanced/pkg/configuration"
	"balanced/pkg/types"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/shlex"
	log "github.com/sirupsen/logrus"
)

const (
	retryAttempts = 3
)

func NewUpdater(cfg *configuration.Config) (*Updater, error) {
	r, err := NewRenderer(cfg.LoadBalancer.Template)
	if err != nil {
		return nil, err
	}

	if cfg.Cloud.AWS == nil {
		return nil, errors.New("awscloud configuration has not been set")
	}

	p, err := awscloud.New(cfg)
	if err != nil {
		return nil, err
	}

	return &Updater{
		cfg:   cfg,
		p:     p,
		r:     r,
		cache: make(map[string]*types.LoadBalancerUpstreamDefinition),
	}, nil
}

type Updater struct {
	cfg            *configuration.Config
	r              *Renderer
	p              cloud.CloudProvider
	cache          map[string]*types.LoadBalancerUpstreamDefinition
	reloadRequired bool
}

func (u *Updater) Start(changes chan *types.Change) {
	ticker := time.NewTicker(*u.cfg.LoadBalancer.ReconcileDuration)
	defer ticker.Stop()

	domains := make([]string, 0)

	for {
		select {
		case change, ok := <-changes:
			if !ok {
				return
			}

			u.cache[change.Obj.Domain] = change.Obj

			if !u.shouldProcessChange(change) {
				changes <- change
				continue
			}

			if err := u.handleChange(change.Obj); err != nil {
				log.Error(err)
				change.Retried += 1
				if change.Retried < retryAttempts {
					log.Infof("retry %d/%d: reschedule change for %s", change.Retried, retryAttempts, change.Obj.Domain)
					change.RetryAfter = aws.Time(time.Now().Add(time.Second * 5))
					changes <- change
				} else {
					log.Infof("retry %d/%d: change for %s could not be applied", change.Retried, retryAttempts, change.Obj.Domain)
				}
				continue
			}

			domains = append(domains, change.Obj.Domain)
			if err := u.p.ReconcileSecurityGroups(map[string]*types.LoadBalancerUpstreamDefinition{"update": change.Obj}, false); err != nil {
				log.Error(err)
			}

		case <-ticker.C:
			if err := u.p.ReconcileSecurityGroups(u.cache, true); err != nil {
				log.Error(err)
			}

			if u.reloadRequired {
				u.reloadRequired = false

				if reloadErr := u.reloadProcess(); reloadErr != nil {
					log.Error(reloadErr)
				}

				log.Debugf("process reloaded successfully")
			}

			if u.cfg.DNS.Enabled {
				if err := u.p.UpsertRecordSet(domains); err != nil {
					log.Error(err)
					// don't empty the domain buffer on error
					continue
				}
			}
			domains = make([]string, 0)
		}
	}
}

func (u *Updater) shouldProcessChange(change *types.Change) bool {
	if change.RetryAfter != nil {
		if time.Now().Before(*change.RetryAfter) {
			return false
		}
	}

	return true
}

func (u *Updater) handleChange(change *types.LoadBalancerUpstreamDefinition) error {
	filename := strings.ReplaceAll(change.Domain, ".", "_") + ".cfg"
	tmpFilePath := filepath.Join("/tmp", filename)

	if tmpErr := u.tryWriteToFile(tmpFilePath, change); tmpErr != nil {
		return tmpErr
	}

	fullFilePath := filepath.Join(u.cfg.LoadBalancer.ConfigDir, filename)

	areEq, err := checksumsEqual(tmpFilePath, fullFilePath)
	if err != nil {
		return err
	}

	if areEq {
		log.Debugf("configuration for %s domain is already up to date, skipping", change.Domain)
		return nil
	}

	log.Debugf("configuration for %s domain has changed, updating", change.Domain)

	if fErr := u.tryWriteToFile(fullFilePath, change); fErr != nil {
		return fErr
	}

	log.Debugf("successfully updated configuration file %s", fullFilePath)
	log.Debug("reload required")
	u.reloadRequired = true

	return nil
}

func (u *Updater) tryWriteToFile(fullFilePath string, change *types.LoadBalancerUpstreamDefinition) error {
	f, fErr := os.OpenFile(fullFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)

	if fErr != nil {
		return fmt.Errorf("unable to open %s: %s", fullFilePath, fErr)
	}

	if wErr := u.r.ToWriter(f, change); wErr != nil {
		return fmt.Errorf("unable to write to file %s: %s", fullFilePath, wErr)
	}

	return nil
}

func (u *Updater) reloadProcess() error {
	cmdParts, err := shlex.Split(u.cfg.LoadBalancer.ReloadCmd)
	if err != nil {
		return err
	}
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	return cmd.Run()
}
