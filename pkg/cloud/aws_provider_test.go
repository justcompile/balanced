package cloud

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/stretchr/testify/assert"
)

func TestAWSProviderGetAddresses(t *testing.T) {
	tests := map[string]struct {
		client            *mockEC2Service
		cfg               *LookupConfig
		expectedAddresses []string
		expectedErr       error
	}{
		"returns error if ec2.DescribeInstances does": {
			&mockEC2Service{responseErr: errors.New("api error")},
			&LookupConfig{},
			nil,
			errors.New("discovery: describing instances failed: api error"),
		},
		"returns empty list if instances do not have public ips and config requires them": {
			&mockEC2Service{
				instances: []*ec2.Instance{
					{PrivateIpAddress: aws.String("10.0.0.1")},
				},
			},
			&LookupConfig{UsePublicIP: true},
			nil,
			nil,
		},
		"returns empty list if instances are ec2 classic": {
			&mockEC2Service{
				instances: []*ec2.Instance{
					{},
				},
			},
			&LookupConfig{},
			nil,
			nil,
		},
		"returns instance public ip addresses": {
			&mockEC2Service{
				instances: []*ec2.Instance{
					{PublicIpAddress: aws.String("10.0.0.1")},
				},
			},
			&LookupConfig{UsePublicIP: true},
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
			&LookupConfig{UsePublicIP: false},
			[]string{"10.1.1.1", "100.1.1.1"},
			nil,
		},
	}

	for name, test := range tests {
		p := &AWSProvider{
			ec2Client: test.client,
		}

		addrs, err := p.GetAddresses(test.cfg)

		assert.Equal(t, test.expectedErr, err, name)
		assert.Equal(t, test.expectedAddresses, addrs, name)
	}
}

func TestAWSProviderUpdateRecords(t *testing.T) {
	tests := map[string]struct {
		client      *mockRoute53
		domains     []string
		expectedErr error
	}{}

	for name, test := range tests {
		p := &AWSProvider{r53Client: test.client}
		err := p.UpsertRecordSet(test.domains)
		assert.Equal(t, test.expectedErr, err, name)
	}
}
