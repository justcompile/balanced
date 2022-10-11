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

	"github.com/google/shlex"
	"github.com/opentracing/opentracing-go/log"
)

func NewUpdater(cfg *configuration.Config) (*Updater, error) {
	r, err := NewRenderer(cfg.LoadBalancer.Template)
	if err != nil {
		return nil, err
	}

	d, err := dns.NewRoute53Updater(&cfg.DNS)
	if err != nil {
		return nil, err
	}

	return &Updater{cfg: cfg.LoadBalancer, d: d, r: r}, nil
}

type Updater struct {
	cfg *configuration.LoadBalancer
	r   *Renderer
	d   dns.Updater
}

func (u *Updater) Start(changes <-chan *types.LoadBalancerUpstreamDefinition) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	addresses := make([]string, 0)

	for {
		select {
		case change, ok := <-changes:
			if !ok {
				return
			}

			if err := u.handleChange(change); err != nil {
				log.Error(err)
				continue
			}

			addresses = append(addresses, change.Domain)

		case <-ticker.C:
			if err := u.d.UpsertRecordSet(addresses); err != nil {
				log.Error(err)
			}
			addresses = make([]string, 0)
		}
	}
}

func (u *Updater) handleChange(change *types.LoadBalancerUpstreamDefinition) error {
	filename := strings.ReplaceAll(change.Domain, ".", "_") + ".cfg"
	fullFilePath := filepath.Join(u.cfg.ConfigDir, filename)

	f, fErr := os.OpenFile(fullFilePath, os.O_RDWR|os.O_CREATE, 0644)

	if fErr != nil {
		return fmt.Errorf("unable to open %s: %s", fullFilePath, fErr)
	}

	if wErr := u.r.ToWriter(f, change); wErr != nil {
		return fmt.Errorf("unable to write to file %s: %s", fullFilePath, wErr)
	}

	if reloadErr := u.reloadProcess(); reloadErr != nil {
		fmt.Errorf("error reloading loadbalancer configuration after update: %s", reloadErr)
	}

	return nil
}

func (u *Updater) reloadProcess() error {
	cmdParts, err := shlex.Split(u.cfg.ReloadCmd)
	if err != nil {
		return err
	}
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	return cmd.Run()
}
