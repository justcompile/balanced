package types

import (
	"bytes"
	"fmt"
	"maps"
	"net"
	"slices"
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

	AddedServers   []*Server
	RemovedServers []*Server
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

func NewLoadBalancerDefinitionChange(domain, healthCheck string, oldEndpoint, newEndpoint *discoveryv1.EndpointSlice) *Change {
	if newEndpoint == nil {
		return nil
	}

	def := &LoadBalancerUpstreamDefinition{
		Domain:         domain,
		HealthCheck:    healthCheck,
		Servers:        make([]*Server, 0),
		AddedServers:   make([]*Server, 0),
		RemovedServers: make([]*Server, 0),
	}

	oldEndpointServers := make(map[string]*Server)
	newEndpointServers := make(map[string]*Server)

	if oldEndpoint != nil {
		port := oldEndpoint.Ports[0].Port
		for _, ss := range oldEndpoint.Endpoints {
			for _, a := range ss.Addresses {
				oldEndpointServers[fmt.Sprintf("%s-%s-%d", ss.TargetRef.Name, a, ptr.Deref(port, 0))] = &Server{
					Id:        ss.TargetRef.Name,
					IPAddress: a,
					Port:      *port,
					Meta: &ServerMeta{
						Hostname: ptr.Deref(ss.Hostname, ""),
						NodeName: ptr.Deref(ss.NodeName, ""),
					},
				}
			}
		}
	}

	for _, ss := range newEndpoint.Endpoints {
		port := newEndpoint.Ports[0].Port

		for _, a := range ss.Addresses {
			s := &Server{
				Id:        ss.TargetRef.Name,
				IPAddress: a,
				Port:      *port,
				Meta: &ServerMeta{
					Hostname: ptr.Deref(ss.Hostname, ""),
					NodeName: ptr.Deref(ss.NodeName, ""),
				},
			}

			def.Servers = append(def.Servers, s)
			newEndpointServers[fmt.Sprintf("%s-%s-%d", ss.TargetRef.Name, a, ptr.Deref(port, 0))] = s
		}
	}

	def.AddedServers = diffServers(newEndpointServers, oldEndpointServers)
	def.RemovedServers = diffServers(oldEndpointServers, newEndpointServers)

	return &Change{Obj: def}
}

func diffServers(a, b map[string]*Server) []*Server {
	ret := make([]*Server, 0)

	aSet := make(Set[string])
	bSet := make(Set[string])

	aSet.Add(slices.Collect(maps.Keys(a))...)
	bSet.Add(slices.Collect(maps.Keys(b))...)

	// Diff returns a new set with elements in the set that are in `aSet` but not `bSet`.
	for k := range aSet.Diff(bSet) {
		ret = append(ret, a[k])
	}

	return ret
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
