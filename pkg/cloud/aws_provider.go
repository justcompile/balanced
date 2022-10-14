package cloud

import (
	"balanced/pkg/configuration"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
)

func getAWSSession(region string) (*session.Session, error) {
	config := aws.Config{
		Region: &region,
		Credentials: credentials.NewChainCredentials(
			[]credentials.Provider{
				&credentials.EnvProvider{},
				&credentials.SharedCredentialsProvider{},
				defaults.RemoteCredProvider(*(defaults.Config()), defaults.Handlers()),
			},
		),
	}

	return session.NewSession(&config)
}

type AWSProvider struct {
	cfg       *configuration.DNS
	lookup    *LookupConfig
	ec2Client ec2iface.EC2API
	r53Client route53iface.Route53API
}

func (a *AWSProvider) GetAddresses(cfg *LookupConfig) ([]string, error) {
	resp, err := a.ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:" + cfg.TagKey),
				Values: []*string{aws.String(cfg.TagValue)},
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String("running")},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("discovery: describing instances failed: %s", err)
	}

	var addrs []string
	for _, r := range resp.Reservations {
		for _, inst := range r.Instances {
			if cfg.UsePublicIP {
				if inst.PublicIpAddress == nil {
					continue
				}

				addrs = append(addrs, *inst.PublicIpAddress)
			} else {
				// EC2-Classic don't have the PrivateIpAddress field
				if inst.PrivateIpAddress == nil {
					continue
				}

				addrs = append(addrs, *inst.PrivateIpAddress)
			}
		}
	}

	return addrs, nil
}
func (a *AWSProvider) UpsertRecordSet(domains []string) error {
	if len(domains) == 0 {
		return nil
	}

	addresses, err := a.GetAddresses(a.lookup)
	if err != nil {
		return err
	}

	records := make([]*route53.ResourceRecord, len(addresses))
	for i, addr := range addresses {
		records[i] = &route53.ResourceRecord{
			Value: aws.String(addr),
		}
	}

	changes := make([]*route53.Change, len(domains))
	for i, domain := range domains {
		changes[i] = &route53.Change{
			Action: aws.String(route53.ChangeActionUpsert),
			ResourceRecordSet: &route53.ResourceRecordSet{
				Name:            aws.String(domain),
				ResourceRecords: records,
				Type:            aws.String(a.cfg.Route53.Type),
				TTL:             aws.Int64(a.cfg.Route53.TTL),
			},
		}
	}

	input := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(a.cfg.Route53.HostedZoneId),
		ChangeBatch: &route53.ChangeBatch{
			Changes: changes,
		},
	}

	_, changeErr := a.r53Client.ChangeResourceRecordSets(input)
	return changeErr
}

func NewAWSProvider(cfg *configuration.DNS) (*AWSProvider, error) {
	ec2meta := ec2metadata.New(session.Must(session.NewSession()))
	tagValue, err := ec2meta.GetMetadata("tags/instance/" + cfg.TagKey)
	if err != nil {
		return nil, fmt.Errorf("route-53: retrieving instance tags failed: %s", err)
	}

	doc, err := ec2meta.GetInstanceIdentityDocument()
	if err != nil {
		return nil, fmt.Errorf("route-53: retrieving instance information failed: %s", err)
	}

	sess, err := getAWSSession(doc.Region)
	if err != nil {
		return nil, fmt.Errorf("aws: unable to initialise session: %s", err)
	}

	p := &AWSProvider{
		cfg: cfg,
		lookup: &LookupConfig{
			TagKey:      cfg.TagKey,
			TagValue:    tagValue,
			UsePublicIP: cfg.UsePublicAddress,
		},
		ec2Client: ec2.New(sess),
		r53Client: route53.New(sess),
	}

	return p, nil
}
