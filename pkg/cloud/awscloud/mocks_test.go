package awscloud

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
)

type mockEC2Service struct {
	instances   []*ec2.Instance
	responseErr error
	ec2iface.EC2API
}

func (m *mockEC2Service) DescribeInstances(*ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	if m.responseErr != nil {
		return nil, m.responseErr
	}

	if len(m.instances) == 0 {
		return nil, nil
	}

	return &ec2.DescribeInstancesOutput{
		Reservations: []*ec2.Reservation{
			{Instances: m.instances},
		},
	}, nil
}

type mockRoute53 struct {
	err         error
	ipsToReturn []string
	route53iface.Route53API
}

func (m *mockRoute53) ChangeResourceRecordSets(*route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error) {
	return nil, m.err
}

func (m *mockRoute53) ListResourceRecordSets(*route53.ListResourceRecordSetsInput) (*route53.ListResourceRecordSetsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	records := make([]*route53.ResourceRecord, len(m.ipsToReturn))

	for i, addr := range m.ipsToReturn {
		records[i] = &route53.ResourceRecord{
			Value: aws.String(addr),
		}
	}

	return &route53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []*route53.ResourceRecordSet{
			{ResourceRecords: records},
		},
	}, nil
}
