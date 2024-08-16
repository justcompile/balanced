package loadbalancer

import (
	"balanced/pkg/configuration"
	"balanced/pkg/dns"
	"balanced/pkg/types"
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

	reg, err := dns.NewCommandRegistrar(&cfg.DNS)
	if err != nil {
		return nil, err
	}

	return &Updater{
		cfg:    cfg,
		dns:    reg,
		render: r,
		cache:  make(map[string]*types.LoadBalancerUpstreamDefinition),
	}, nil
}

type Updater struct {
	cfg            *configuration.Config
	render         *Renderer
	dns            dns.Registrar
	cache          map[string]*types.LoadBalancerUpstreamDefinition
	reloadRequired bool
}

func (u *Updater) OnExit() error {
	if u.dns != nil {
		return u.dns.RemoveAll()
	}

	return nil
}

func (u *Updater) Start(changes chan *types.Change) {
	ticker := time.NewTicker(*u.cfg.LoadBalancer.ReconcileDuration)
	defer ticker.Stop()

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

			if err := u.dns.Add(change.Obj.Domain); err != nil {
				log.Errorf("unable to update DNS record for %s: %s", change.Obj.Domain, err)
			}
		case <-ticker.C:
			if u.reloadRequired {
				u.reloadRequired = false

				if reloadErr := u.reloadProcess(); reloadErr != nil {
					log.Error(reloadErr)
				}

				log.Debugf("process reloaded successfully")
			}
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

	if wErr := u.render.ToWriter(f, change); wErr != nil {
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
