package awscloud

import (
	"balanced/pkg/configuration"
	"balanced/pkg/types"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	log "github.com/sirupsen/logrus"
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
	cfg       *configuration.AWS
	dnsCfg    *configuration.DNS
	metaData  *instanceMetaData
	ec2Client ec2iface.EC2API
	r53Client route53iface.Route53API
}

func (a *AWSProvider) GetAddresses() ([]string, error) {
	resp, err := a.ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{a.metaData.instanceID}),
	})
	if err != nil {
		return nil, fmt.Errorf("discovery: describing instances failed: %s", err)
	}

	var addrs []string
	for _, r := range resp.Reservations {
		for _, inst := range r.Instances {
			if a.dnsCfg.UsePublicAddress {
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

	addresses, err := a.GetAddresses()
	if err != nil {
		return err
	}

	changes := make([]*route53.Change, 0)
	for _, domain := range domains {
		recordSet, err := a.recordSetForUpdate(domain, addresses)
		if err != nil {
			return err
		}

		if recordSet == nil {
			log.Infof("no new addresses detected for DNS record %s, skipping", domain)
			continue
		}

		changes = append(changes, &route53.Change{
			Action:            aws.String(route53.ChangeActionUpsert),
			ResourceRecordSet: recordSet,
		})
	}

	if len(changes) == 0 {
		return nil
	}

	input := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(a.cfg.HostedZoneId),
		ChangeBatch: &route53.ChangeBatch{
			Changes: changes,
		},
	}

	log.Debugf("making DNS changes: %v", input)

	_, changeErr := a.r53Client.ChangeResourceRecordSets(input)
	return changeErr
}

func (a *AWSProvider) recordSetForUpdate(domain string, addressesToAdd []string) (*route53.ResourceRecordSet, error) {
	if len(addressesToAdd) == 0 {
		log.Debugf("no addresses supplied to update %s, skipping", domain)
		return nil, nil
	}

	log.Debugf("looking up route53 record for %s", domain)

	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(a.cfg.HostedZoneId),
		StartRecordName: aws.String(domain),
		StartRecordType: aws.String(a.cfg.Type),
		MaxItems:        aws.String("1"),
	}

	resp, err := a.r53Client.ListResourceRecordSets(input)
	if err != nil {
		return nil, fmt.Errorf("unable to locate resource records for domain %s: %s", domain, err)
	}

	log.Debugf("found record set for %s: %v", domain, resp.ResourceRecordSets)

	var recordSet *route53.ResourceRecordSet

	if len(resp.ResourceRecordSets) == 0 {
		recordSet = &route53.ResourceRecordSet{
			Name:            aws.String(domain),
			Type:            aws.String(a.cfg.Type),
			ResourceRecords: make([]*route53.ResourceRecord, 0),
			TTL:             aws.Int64(defaultRecordSetTTL),
		}
	} else {
		recordSet = resp.ResourceRecordSets[0]
	}

	existing := make(types.Set[string])
	values := make(types.Set[string])

	for _, v := range recordSet.ResourceRecords {
		values.Add(*v.Value)
		existing.Add(*v.Value)
	}
	values.Add(addressesToAdd...)

	if len(existing.Diff(values)) == 0 && len(values.Diff(existing)) == 0 {
		log.Debugf("no DNS changes discovered for %s", domain)
		return nil, nil
	}

	log.Debugf("DNS %s differs. current: %v, desired: %v", domain, existing, values)

	recordSet.ResourceRecords = make([]*route53.ResourceRecord, len(values))

	i := 0
	for addr := range values {
		recordSet.ResourceRecords[i] = &route53.ResourceRecord{
			Value: aws.String(addr),
		}
		i++
	}

	return recordSet, nil
}

func New(cfg *configuration.Config) (*AWSProvider, error) {
	meta, err := getInstanceMetaData()
	if err != nil {
		return nil, err
	}

	sess, err := getAWSSession(meta.region)
	if err != nil {
		return nil, fmt.Errorf("aws: unable to initialise session: %s", err)
	}

	p := &AWSProvider{
		cfg:       cfg.Cloud.AWS,
		dnsCfg:    &cfg.DNS,
		metaData:  meta,
		ec2Client: ec2.New(sess),
		r53Client: route53.New(sess),
	}

	return p, nil
}
