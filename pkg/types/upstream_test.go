package types

import (
	"net"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestLoadBalancerUpstreamDefinitionFromK8sEndpoint(t *testing.T) {
	domain := "foo.com"
	tests := map[string]struct {
		endpoint *corev1.Endpoints
		expected *Change
	}{
		"returns nil when endpoints is nil": {
			nil,
			nil,
		},
		"returns definition for endpoint": {
			&corev1.Endpoints{
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{IP: "10.1.1.1", TargetRef: &corev1.ObjectReference{Name: "my-pod-1"}, NodeName: aws.String("node-1")},
						},
						Ports: []corev1.EndpointPort{{Port: 8443}},
					},
				},
			},
			&Change{
				Obj: &LoadBalancerUpstreamDefinition{
					Domain: domain,
					Servers: []*Server{
						{Id: "my-pod-1", IPAddress: "10.1.1.1", Port: 8443, Meta: &ServerMeta{NodeName: "node-1"}},
					},
				},
			},
		},
	}

	for name, test := range tests {
		change := NewLoadBalancerDefinitionChange(domain, test.endpoint)
		assert.Equal(t, test.expected, change, name)
	}
}

func TestSortedIPsFromEndpoint(t *testing.T) {
	tests := map[string]struct {
		endpoint *corev1.Endpoints
		expected []net.IP
	}{
		"returns nil when endpoint is nil": {
			nil,
			nil,
		},
		"returns single ip when endpoint only maps to one address": {
			&corev1.Endpoints{
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{IP: "10.1.1.1", TargetRef: &corev1.ObjectReference{Name: "my-pod-1"}},
						},
						Ports: []corev1.EndpointPort{{Port: 8443}},
					},
				},
			},
			[]net.IP{
				net.ParseIP("10.1.1.1"),
			},
		},
		"returns ips ordered ascendingly": {
			&corev1.Endpoints{
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{IP: "10.1.1.1", TargetRef: &corev1.ObjectReference{Name: "my-pod-1"}},
						},
						Ports: []corev1.EndpointPort{{Port: 8443}},
					},
					{
						Addresses: []corev1.EndpointAddress{
							{IP: "10.1.2.1", TargetRef: &corev1.ObjectReference{Name: "my-pod-1"}},
						},
						Ports: []corev1.EndpointPort{{Port: 8443}},
					},
					{
						Addresses: []corev1.EndpointAddress{
							{IP: "10.10.1.10", TargetRef: &corev1.ObjectReference{Name: "my-pod-1"}},
						},
						Ports: []corev1.EndpointPort{{Port: 8443}},
					},
					{
						Addresses: []corev1.EndpointAddress{
							{IP: "10.10.1.1", TargetRef: &corev1.ObjectReference{Name: "my-pod-1"}},
						},
						Ports: []corev1.EndpointPort{{Port: 8443}},
					},
				},
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
