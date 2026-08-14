package dataplane

import "balanced/pkg/types"

/*
config-dir = "/etc/haproxy/conf.d"
reload-cmd = "systemctl reload haproxy"
template = """
backend {{.Domain}}

	http-check send meth GET uri {{.HealthCheck}} hdr Host {{.Domain}}
	balance roundrobin
	{{range .Servers -}}
	server {{.Id}} {{.IPAddress}}:{{.Port}} check check-ssl
	{{end}}
*/
func BackendFromUpstream(def *types.LoadBalancerUpstreamDefinition) *Backend {
	return &Backend{
		Balance: Balance{
			Algorithm: "roundrobin",
		},
		Name: def.Domain,
		HTTPCheck: HTTPCheck{
			Headers: []Header{
				{Key: "Host", Value: def.Domain},
			},
			Method: "GET",
			Type:   "send",
			Uri:    def.HealthCheck,
		},
	}
}

func ServersFromUpstream(def *types.LoadBalancerUpstreamDefinition) ([]*Server, []*Server, []string) {
	allServers := make([]*Server, len(def.Servers))

	for i, s := range def.Servers {
		allServers[i] = k8sServerToHAProxyServer(s)
	}

	addedServers := make([]*Server, len(def.AddedServers))
	for i, s := range def.AddedServers {
		addedServers[i] = k8sServerToHAProxyServer(s)
	}

	removedServers := make([]string, len(def.RemovedServers))
	for i, s := range def.RemovedServers {
		removedServers[i] = s.Id
	}

	return allServers, addedServers, removedServers
}

func k8sServerToHAProxyServer(s *types.Server) *Server {
	return &Server{
		Check:      "enabled",
		CheckSSL:   "enabled",
		Identifier: s.Id,
		Address:    s.IPAddress,
		Port:       int(s.Port),
	}
}
