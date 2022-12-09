package awscloud

import (
	"balanced/pkg/configuration"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/stretchr/testify/assert"
)

func TestAWSProviderGetAddresses(t *testing.T) {
	tests := map[string]struct {
		client            *mockEC2Service
		usePublicIP       bool
		expectedAddresses []string
		expectedErr       error
	}{
		"returns error if ec2.DescribeInstances does": {
			&mockEC2Service{responseErr: errors.New("api error")},
			false,
			nil,
			errors.New("discovery: describing instances failed: api error"),
		},
		"returns empty list if instances do not have public ips and config requires them": {
			&mockEC2Service{
				instances: []*ec2.Instance{
					{PrivateIpAddress: aws.String("10.0.0.1")},
				},
			},
			true,
			nil,
			nil,
		},
		"returns empty list if instances are ec2 classic": {
			&mockEC2Service{
				instances: []*ec2.Instance{
					{},
				},
			},
			true,
			nil,
			nil,
		},
		"returns instance public ip addresses": {
			&mockEC2Service{
				instances: []*ec2.Instance{
					{PublicIpAddress: aws.String("10.0.0.1")},
				},
			},
			true,
			[]string{"10.0.0.1"},
			nil,
		},
		"returns instance private ip addresses": {
			&mockEC2Service{
				instances: []*ec2.Instance{
					{PublicIpAddress: aws.String("3.0.0.1"), PrivateIpAddress: aws.String("10.1.1.1")},
					{PublicIpAddress: aws.String("4.0.0.1"), PrivateIpAddress: aws.String("100.1.1.1")},
				},
			},
			false,
			[]string{"10.1.1.1", "100.1.1.1"},
			nil,
		},
	}

	for name, test := range tests {
		p := &AWSProvider{
			ec2Client: test.client,
			dnsCfg:    &configuration.DNS{UsePublicAddress: test.usePublicIP},
			metaData:  &instanceMetaData{instanceID: "i-12345"},
		}

		addrs, err := p.GetAddresses()

		assert.Equal(t, test.expectedErr, err, name)
		assert.Equal(t, test.expectedAddresses, addrs, name)
	}
}

func TestAWSProviderUpdateRecords(t *testing.T) {
	tests := map[string]struct {
		r53            *mockRoute53
		domain         string
		instanceIPs    []string
		expectedRecord *route53.ResourceRecordSet
		expectedErr    error
	}{
		"returns nil when no addresses are supplied": {
			&mockRoute53{err: errors.New("no recordsets found")},
			"foo.com",
			nil,
			nil,
			nil,
		},
		"returns error when error occurs during list call": {
			&mockRoute53{err: errors.New("no recordsets found")},
			"foo.com",
			[]string{"10.1.1.1"},
			nil,
			errors.New("unable to locate resource records for domain foo.com: no recordsets found"),
		},
		"returns new record with supplied values if record does not exist": {
			&mockRoute53{},
			"foo.com",
			[]string{"10.1.1.1"},
			&route53.ResourceRecordSet{
				Name: aws.String("foo.com"),
				Type: aws.String(""),
				TTL:  aws.Int64(defaultRecordSetTTL),
				ResourceRecords: []*route53.ResourceRecord{
					{Value: aws.String("10.1.1.1")},
				},
			},
			nil,
		},
		"returns nil record with when no changes are required": {
			&mockRoute53{hostname: "foo.com", ipsToReturn: []string{"10.1.1.1", "10.1.1.2"}},
			"foo.com",
			[]string{"10.1.1.1"},
			nil,
			nil,
		},
		"returns record with newly added values values": {
			&mockRoute53{hostname: "foo.com", ipsToReturn: []string{"10.1.1.1"}},
			"foo.com",
			[]string{"10.1.1.2"},
			&route53.ResourceRecordSet{
				Name: aws.String("foo.com"),
				Type: aws.String("A"),
				TTL:  aws.Int64(defaultRecordSetTTL),
				ResourceRecords: []*route53.ResourceRecord{
					{Value: aws.String("10.1.1.1")},
					{Value: aws.String("10.1.1.2")},
				},
			},
			nil,
		},
	}

	for name, test := range tests {
		p := &AWSProvider{
			cfg:       &configuration.AWS{},
			dnsCfg:    &configuration.DNS{},
			r53Client: test.r53,
		}
		addrs, err := p.recordSetForUpdate(test.domain, test.instanceIPs)
		assert.Equal(t, test.expectedErr, err, name)
		assert.Equal(t, test.expectedRecord, addrs, name)
	}
}
