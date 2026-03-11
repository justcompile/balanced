package types

import (
	"bytes"
	"net"
	"sort"
	"time"

	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/utils/ptr"
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

func NewLoadBalancerDefinitionChange(domain, healthCheck string, endpoint *discoveryv1.EndpointSlice) *Change {
	if endpoint == nil {
		return nil
	}

	def := &LoadBalancerUpstreamDefinition{
		Domain:      domain,
		HealthCheck: healthCheck,
		Servers:     make([]*Server, 0),
	}

	for _, ss := range endpoint.Endpoints {
		port := endpoint.Ports[0].Port

		for _, a := range ss.Addresses {
			def.Servers = append(def.Servers, &Server{
				Id:        ss.TargetRef.Name,
				IPAddress: a,
				Port:      *port,
				Meta: &ServerMeta{
					Hostname: ptr.Deref(ss.Hostname, ""),
					NodeName: ptr.Deref(ss.NodeName, ""),
				},
			})
		}
	}

	return &Change{Obj: def}
}

func SortedIPsFromEndpoint(e *discoveryv1.EndpointSlice) []net.IP {
	if e == nil {
		return nil
	}
	addresses := make([]net.IP, 0)

	for _, ss := range e.Endpoints {
		for _, a := range ss.Addresses {
			addresses = append(addresses, net.ParseIP(a))
		}
	}

	sort.Slice(addresses, func(i, j int) bool {
		return bytes.Compare(addresses[i], addresses[j]) < 0
	})

	return addresses
}
