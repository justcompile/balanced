package loadbalancer

import (
	"balanced/pkg/configuration"
	"balanced/pkg/types"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/shlex"
	log "github.com/sirupsen/logrus"
)

func NewUpdater(cfg *configuration.LoadBalancer) (*Updater, error) {
	r, err := NewRenderer(cfg.Template)
	if err != nil {
		return nil, err
	}

	return &Updater{cfg: cfg, r: r}, nil
}

type Updater struct {
	cfg *configuration.LoadBalancer
	r   *Renderer
}

func (u *Updater) Start(changes <-chan *types.LoadBalancerUpstreamDefinition) {
	for {
		change, ok := <-changes
		if !ok {
			return
		}

		filename := strings.ReplaceAll(change.Domain, ".", "_") + ".cfg"
		fullFilePath := filepath.Join(u.cfg.ConfigDir, filename)

		f, fErr := os.OpenFile(fullFilePath, os.O_RDWR|os.O_CREATE, 0644)

		if fErr != nil {
			log.Errorf("unable to open %s: %s", fullFilePath, fErr)
			continue
		}

		if wErr := u.r.ToWriter(f, change); wErr != nil {
			log.Errorf("unable to write to file %s: %s", fullFilePath, wErr)
			continue
		}

		if reloadErr := u.reloadProcess(); reloadErr != nil {
			log.Errorf("error reloading loadbalancer configuration after update: %s", reloadErr)
		}
	}
}

func (u *Updater) reloadProcess() error {
	cmdParts, err := shlex.Split(u.cfg.ReloadCmd)
	if err != nil {
		return err
	}
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	return cmd.Run()
}
