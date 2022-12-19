package types

import (
	"bytes"
	"net"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
)

type Change struct {
	Obj        *LoadBalancerUpstreamDefinition
	Retried    int
	RetryAfter *time.Time
}

type LoadBalancerUpstreamDefinition struct {
	Domain      string
	HealthCheck string
	Servers     []*Server
}

type Server struct {
	Id        string
	IPAddress string
	Port      int32
	Meta      *ServerMeta
}

type ServerMeta struct {
	Hostname string
	NodeName string
}

func NewLoadBalancerDefinitionChange(domain, healthCheck string, endpoint *corev1.Endpoints) *Change {
	if endpoint == nil {
		return nil
	}

	def := &LoadBalancerUpstreamDefinition{
		Domain:      domain,
		HealthCheck: healthCheck,
		Servers:     make([]*Server, 0),
	}

	for _, ss := range endpoint.Subsets {
		port := ss.Ports[0].Port

		for _, a := range ss.Addresses {
			def.Servers = append(def.Servers, &Server{
				Id:        a.TargetRef.Name,
				IPAddress: a.IP,
				Port:      port,
				Meta: &ServerMeta{
					Hostname: a.Hostname,
					NodeName: *a.NodeName,
				},
			})
		}
	}

	return &Change{Obj: def}
}

func SortedIPsFromEndpoint(e *corev1.Endpoints) []net.IP {
	if e == nil {
		return nil
	}
	addresses := make([]net.IP, 0)

	for _, ss := range e.Subsets {
		for _, a := range ss.Addresses {
			addresses = append(addresses, net.ParseIP(a.IP))
		}
	}

	sort.Slice(addresses, func(i, j int) bool {
		return bytes.Compare(addresses[i], addresses[j]) < 0
	})

	return addresses
}
