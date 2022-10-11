package dns

import (
	"balanced/pkg/configuration"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/hashicorp/go-discover"
)

type Route53Updater struct {
	cfg          *configuration.DNS
	discoverConf *discover.Config
}

func (r *Route53Updater) GetDiscoveryQuery() string {
	return r.discoverConf.String()
}
func (r *Route53Updater) UpsertRecordSet(domains []string) error {
	if len(domains) == 0 {
		return nil
	}

	d := &discover.Discover{}

	addresses, err := d.Addrs(r.GetDiscoveryQuery(), nil)
	if err != nil {
		return err
	}

	r53 := route53.New(session.New())

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
				Type:            aws.String(r.cfg.Route53.Type),
				TTL:             aws.Int64(r.cfg.Route53.TTL),
			},
		}
	}

	input := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(r.cfg.Route53.HostedZoneId),
		ChangeBatch: &route53.ChangeBatch{
			Changes: changes,
		},
	}

	_, changeErr := r53.ChangeResourceRecordSets(input)
	return changeErr
}

func NewRoute53Updater(cfg *configuration.DNS) (*Route53Updater, error) {
	if cfg.Route53 == nil {
		return nil, fmt.Errorf("dns.route53 configuration has not been set")
	}

	ec2meta := ec2metadata.New(session.New())
	identity, err := ec2meta.GetInstanceIdentityDocument()
	if err != nil {
		return nil, fmt.Errorf("route-53: GetInstanceIdentityDocument failed: %s", err)
	}

	tagValue, err := ec2meta.GetMetadata("tags/instance/" + cfg.TagKey)
	if err != nil {
		return nil, fmt.Errorf("route-53: retrieving instance tags failed: %s", err)
	}

	discoverConf := discover.Config{
		"provider":  "aws",
		"region":    identity.Region,
		"tag_key":   cfg.TagKey,
		"tag_value": tagValue,
	}

	// defaults to private_v4 so only set if we want to override default behaviour
	if cfg.UsePublicAddress {
		discoverConf["addr_type"] = "public_v4"
	}

	return &Route53Updater{
		cfg:          cfg,
		discoverConf: &discoverConf,
	}, nil
}
