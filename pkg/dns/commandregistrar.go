package dns

import (
	"balanced/pkg/configuration"
	"balanced/pkg/types"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/google/shlex"
	log "github.com/sirupsen/logrus"
)

type CommandRegistrar struct {
	address       string
	addCommand    *template.Template
	removeCommand *template.Template
	knownDomains  types.Set[string]
}

func (c *CommandRegistrar) Add(domain string) error {
	if c.knownDomains.Has(domain) {
		log.Debugf("already know about %s, no action", domain)
		return nil
	}

	if err := c.executeTemplate(c.addCommand, domain); err != nil {
		return err
	}

	c.knownDomains.Add(domain)

	return nil
}

func (c *CommandRegistrar) Remove(domain string) error {
	if !c.knownDomains.Has(domain) {
		return nil
	}

	if err := c.executeTemplate(c.removeCommand, domain); err != nil {
		return err
	}

	c.knownDomains.Remove(domain)

	return nil
}

func (c *CommandRegistrar) RemoveAll() error {
	errors := make([]string, 0)
	for domain := range c.knownDomains {
		if err := c.executeTemplate(c.removeCommand, domain); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("one or more errors occurred: %s", strings.Join(errors, "\n"))
	}

	return nil
}

func (c *CommandRegistrar) executeTemplate(t *template.Template, domain string) error {
	var buf bytes.Buffer
	if err := t.Execute(&buf, map[string]string{"domain": domain, "address": c.address}); err != nil {
		return fmt.Errorf("unable to parse command string: %s", err.Error())
	}

	return run(buf.String())
}

func run(command string) error {
	log.Debugf("executing: %s", command)

	parts, err := shlex.Split(command)
	if err != nil {
		return err
	}

	proc := exec.Command(parts[0], parts[1:]...)
	proc.Env = os.Environ()
	out, err := proc.CombinedOutput()
	log.Debugln(string(out))

	return err
}

func NewCommandRegistrar(cfg *configuration.DNS) (*CommandRegistrar, error) {
	c := &CommandRegistrar{
		address:      cfg.Address,
		knownDomains: make(types.Set[string]),
	}

	if cfg.Custom == nil {
		return nil, errors.New("dns.custom not set in config")
	}

	var err error

	c.addCommand, err = template.New("addCommand").Parse(cfg.Custom.AddCommand)
	if err != nil {
		return nil, fmt.Errorf("unable to parse dns.custom.add-command: %s", err.Error())
	}

	c.removeCommand, err = template.New("removeCommand").Parse(cfg.Custom.RemoveCommand)
	if err != nil {
		return nil, fmt.Errorf("unable to parse dns.custom.remove-command: %s", err.Error())
	}

	return c, nil
}
