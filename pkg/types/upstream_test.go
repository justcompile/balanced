package types

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/utils/ptr"
)

func TestLoadBalancerUpstreamDefinitionFromK8sEndpoint(t *testing.T) {
	domain := "foo.com"
	healthCheck := "/health"

	tests := map[string]struct {
		endpoint *discoveryv1.EndpointSlice
		expected *Change
	}{
		"returns nil when endpoints is nil": {
			nil,
			nil,
		},
		"returns definition for endpoint": {
			&discoveryv1.EndpointSlice{
				Endpoints: []discoveryv1.Endpoint{
					{
						TargetRef: &corev1.ObjectReference{Name: "my-pod-1"}, NodeName: ptr.To("node-1"),
						Addresses: []string{"10.1.1.1"},
					},
				},
				Ports: []discoveryv1.EndpointPort{{Port: ptr.To(int32(8443))}},
			},
			&Change{
				Obj: &LoadBalancerUpstreamDefinition{
					Domain:      domain,
					HealthCheck: healthCheck,
					Servers: []*Server{
						{Id: "my-pod-1", IPAddress: "10.1.1.1", Port: 8443, Meta: &ServerMeta{NodeName: "node-1"}},
					},
				},
			},
		},
	}

	for name, test := range tests {
		change := NewLoadBalancerDefinitionChange(domain, healthCheck, test.endpoint)
		assert.Equal(t, test.expected, change, name)
	}
}

func TestSortedIPsFromEndpoint(t *testing.T) {
	tests := map[string]struct {
		endpoint *discoveryv1.EndpointSlice
		expected []net.IP
	}{
		"returns nil when endpoint is nil": {
			nil,
			nil,
		},
		"returns single ip when endpoint only maps to one address": {
			&discoveryv1.EndpointSlice{
				Endpoints: []discoveryv1.Endpoint{
					{
						TargetRef: &corev1.ObjectReference{Name: "my-pod-1"},
						Addresses: []string{"10.1.1.1"},
					},
				},

				Ports: []discoveryv1.EndpointPort{{Port: ptr.To(int32(8443))}},
			},
			[]net.IP{
				net.ParseIP("10.1.1.1"),
			},
		},
		"returns ips ordered ascendingly": {
			&discoveryv1.EndpointSlice{
				Endpoints: []discoveryv1.Endpoint{
					{
						TargetRef: &corev1.ObjectReference{Name: "my-pod-1"},
						Addresses: []string{"10.1.1.1"},
					},
					{
						TargetRef: &corev1.ObjectReference{Name: "my-pod-1"},
						Addresses: []string{"10.1.2.1"},
					},
					{
						TargetRef: &corev1.ObjectReference{Name: "my-pod-1"},
						Addresses: []string{"10.10.1.10"},
					},
					{
						TargetRef: &corev1.ObjectReference{Name: "my-pod-1"},
						Addresses: []string{"10.10.1.1"},
					},
				},
				Ports: []discoveryv1.EndpointPort{{Port: ptr.To(int32(8443))}},
			},
			[]net.IP{
				net.ParseIP("10.1.1.1"),
				net.ParseIP("10.1.2.1"),
				net.ParseIP("10.10.1.1"),
				net.ParseIP("10.10.1.10"),
			},
		},
	}

	for name, test := range tests {
		actual := SortedIPsFromEndpoint(test.endpoint)
		assert.Equal(t, test.expected, actual, name)
	}
}
