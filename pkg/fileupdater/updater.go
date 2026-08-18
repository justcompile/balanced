package fileupdater

import (
	"balanced/pkg/configuration"
	"balanced/pkg/dns"
	"balanced/pkg/types"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/shlex"
	log "github.com/sirupsen/logrus"
)

type Updater struct {
	cfg            *configuration.Config
	render         *Renderer
	dns            dns.Registrar
	reloadRequired bool
}

func (u *Updater) IsReloadRequired() bool {
	return u.reloadRequired
}

func (u *Updater) ReloadProcess() error {
	u.reloadRequired = false

	cmdParts, err := shlex.Split(u.cfg.LoadBalancer.ReloadCmd)
	if err != nil {
		return err
	}
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	return cmd.Run()
}

func (u *Updater) Update(change *types.LoadBalancerUpstreamDefinition) error {
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

func NewUpdater(cfg *configuration.Config) (*Updater, error) {
	r, err := NewRenderer(cfg.LoadBalancer.Template)
	if err != nil {
		return nil, err
	}

	reg, err := dns.NewDNSRegistrar(&cfg.DNS)
	if err != nil {
		return nil, err
	}

	return &Updater{
		cfg:    cfg,
		dns:    reg,
		render: r,
	}, nil
}
