package loadbalancer

import (
	"balanced/pkg/configuration"
	"balanced/pkg/types"
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

type setupChangeTestHandler func(*configuration.LoadBalancer, *types.LoadBalancerUpstreamDefinition)

func testRenderTemplate(text string, obj interface{}) string {
	t := template.Must(template.New("foobar").Parse(text))

	b := new(strings.Builder)

	if err := t.Execute(b, obj); err != nil {
		return err.Error()
	}

	return b.String()
}

func testReadFile(fp string) string {
	data, err := ioutil.ReadFile(fp)
	if err != nil {
		return err.Error()
	}

	return string(data)
}

func TestUpdater_handleChange(t *testing.T) {
	templateText := `backend {{.Domain}}
  balance roundrobin
  {{range .Servers -}}
  server {{.Id}} {{.IPAddress}}:{{.Port}} check check-ssl
  {{end}}`

	servers := []*types.Server{
		{Id: "one", IPAddress: "10.1.1.1", Port: 80},
		{Id: "two", IPAddress: "10.1.1.2", Port: 80},
		{Id: "three", IPAddress: "10.1.1.3", Port: 80},
	}

	tests := map[string]struct {
		cfg         *configuration.LoadBalancer
		change      *types.LoadBalancerUpstreamDefinition
		setup       setupChangeTestHandler
		expectedErr error
		verify      func(string)
	}{
		"returns error when file path does not exist": {
			&configuration.LoadBalancer{Template: templateText, ReloadCmd: "ls -al", ConfigDir: "/foob"},
			&types.LoadBalancerUpstreamDefinition{Domain: "hi.com"},
			func(*configuration.LoadBalancer, *types.LoadBalancerUpstreamDefinition) {},
			errors.New("unable to open /foob/hi_com.cfg: open /foob/hi_com.cfg: no such file or directory"),
			func(string) {},
		},
		"creates file if it does not exist and populates": {
			&configuration.LoadBalancer{Template: templateText, ReloadCmd: "ls -al", ConfigDir: "/tmp"},
			&types.LoadBalancerUpstreamDefinition{Domain: "hi.com", Servers: servers},
			func(*configuration.LoadBalancer, *types.LoadBalancerUpstreamDefinition) {},
			nil,
			func(name string) {
				fp := "/tmp/hi_com.cfg"
				defer os.Remove(fp)

				assert.Equal(t, testRenderTemplate(templateText, &types.LoadBalancerUpstreamDefinition{Domain: "hi.com", Servers: servers}), testReadFile(fp), name)
			},
		},
		"updates existing file": {
			&configuration.LoadBalancer{Template: templateText, ReloadCmd: "ls -al", ConfigDir: "/tmp"},
			&types.LoadBalancerUpstreamDefinition{Domain: "hi.com", Servers: servers[2:]},
			func(lb *configuration.LoadBalancer, def *types.LoadBalancerUpstreamDefinition) {
				f, err := os.Create(lb.ConfigDir + "/hi_com.cfg")
				if err != nil {
					t.Fatal(err)
				}

				f.WriteString(testRenderTemplate(templateText, def))
			},
			nil,
			func(name string) {
				fp := "/tmp/hi_com.cfg"
				defer os.Remove(fp)

				assert.Equal(t, testRenderTemplate(templateText, &types.LoadBalancerUpstreamDefinition{Domain: "hi.com", Servers: servers[2:]}), testReadFile(fp), name)
			},
		},
	}

	for name, test := range tests {
		test.setup(test.cfg, test.change)

		u := &Updater{
			cfg:    &configuration.Config{LoadBalancer: test.cfg},
			render: &Renderer{t: template.Must(template.New("foo").Parse(templateText))},
		}

		err := u.handleChange(test.change)

		assert.Equal(t, test.expectedErr, err, name)
		test.verify(name)
	}
}
