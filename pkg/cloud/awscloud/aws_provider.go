package awscloud

import (
	"balanced/pkg/cloud"
	"balanced/pkg/configuration"
	"balanced/pkg/types"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/google/uuid"
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
	cfg       *configuration.DNS
	lookup    *cloud.LookupConfig
	ec2Client ec2iface.EC2API
	r53Client route53iface.Route53API
}

func (a *AWSProvider) GetAddresses(cfg *cloud.LookupConfig) ([]string, error) {
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

func (a *AWSProvider) ReconcileSecurityGroups(defs map[string]*types.LoadBalancerUpstreamDefinition, fullSync bool) error {
	ports := types.Set[int64]{}
	nodes := types.Set[string]{}

	for _, def := range defs {
		ports.Add(int64(def.Servers[0].Port))
		nodes.Add(def.Servers[0].Meta.NodeName)
	}

	instances, insErr := a.getInstancesFromDNSNames(nodes)
	if insErr != nil {
		return insErr
	}

	secGroup, sGrpErr := a.upsertSecurityGroupRules(ports, "", instances[0].VpcId, fullSync)
	if sGrpErr != nil {
		return sGrpErr
	}

	return a.associateSecurityGroupToInstances(secGroup, instances)
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

func (a *AWSProvider) upsertSecurityGroupRules(ports types.Set[int64], lbSecurityGroup string, vpcId *string, fullSync bool) (*cloud.SecurityGroup, error) {
	resp, err := a.ec2Client.DescribeSecurityGroups(
		&ec2.DescribeSecurityGroupsInput{
			Filters: []*ec2.Filter{
				{Name: aws.String("tag-key"), Values: aws.StringSlice([]string{cloud.SecurityGroupTag})},
				{Name: aws.String("vpc-id"), Values: []*string{vpcId}},
			},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("awscloud: error discovering security groups: %s", err)
	}

	if len(resp.SecurityGroups) == 0 {
		return a.createSecurityGroup(ports, lbSecurityGroup, vpcId)
	}

	if len(resp.SecurityGroups) > 1 {
		return nil, fmt.Errorf("awscloud: multiple security groups with the tag %s exist", cloud.SecurityGroupTag)
	}

	if fullSync {
		err = a.updateRules(resp.SecurityGroups[0], ports, lbSecurityGroup)
	}

	return &cloud.SecurityGroup{Id: aws.StringValue(resp.SecurityGroups[0].GroupId)}, err
}

func (a *AWSProvider) updateRules(grp *ec2.SecurityGroup, requiredPorts types.Set[int64], destinationGroupId string) error {

	existingPorts := types.Set[int64]{}
	for _, perm := range grp.IpPermissions {
		existingPorts.Add(*perm.FromPort)
	}

	portsToAdd := requiredPorts.Diff(existingPorts)
	portsToRemove := existingPorts.Diff(requiredPorts)
	if len(portsToRemove) > 0 {
		log.Infof("awscloud: removing ports %v from security group %s", portsToRemove, *grp.GroupId)
		if _, err := a.ec2Client.RevokeSecurityGroupIngress(&ec2.RevokeSecurityGroupIngressInput{
			IpPermissions: ipPermissionsFromPorts(portsToRemove, destinationGroupId),
			GroupId:       grp.GroupId,
		}); err != nil {
			return fmt.Errorf("awscloud: an error occured removing ingress rules: %s", err)
		}
	}

	if len(portsToAdd) > 0 {
		log.Infof("awscloud: adding ports %v from security group %s", portsToAdd, *grp.GroupId)

		input := &ec2.AuthorizeSecurityGroupIngressInput{
			IpPermissions: ipPermissionsFromPorts(portsToAdd, destinationGroupId),
			GroupId:       grp.GroupId,
		}

		json.NewEncoder(os.Stdout).Encode(input)

		if _, err := a.ec2Client.AuthorizeSecurityGroupIngress(input); err != nil {
			return fmt.Errorf("awscloud: an error occured updating ingress rules: %s", err)
		}
	}

	return nil
}

func (a *AWSProvider) createSecurityGroup(ports types.Set[int64], lbSecurityGroupId string, vpcId *string) (*cloud.SecurityGroup, error) {
	suffix := strings.Split(uuid.New().String(), "-")[0]
	resp, err := a.ec2Client.CreateSecurityGroup(
		&ec2.CreateSecurityGroupInput{
			Description: aws.String("Ingress Rules from Balanced Load Balancer"),
			GroupName:   aws.String("balanced-to-eks-ingress-" + suffix),
			VpcId:       vpcId,
			TagSpecifications: []*ec2.TagSpecification{
				(&ec2.TagSpecification{}).
					SetResourceType("security-group").
					SetTags([]*ec2.Tag{
						(&ec2.Tag{}).SetKey(cloud.SecurityGroupTag).SetValue("1"),
					}),
			},
		},
	)
	if err != nil {
		return nil, err
	}

	permissions := ipPermissionsFromPorts(ports, lbSecurityGroupId)

	_, rulesErr := a.ec2Client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId:       resp.GroupId,
		IpPermissions: permissions,
	})

	if err != nil {
		return nil, rulesErr
	}

	return &cloud.SecurityGroup{Id: aws.StringValue(resp.GroupId)}, nil
}

func (a *AWSProvider) associateSecurityGroupToInstances(secGroup *cloud.SecurityGroup, instances []*ec2.Instance) error {
	for _, instance := range instances {
		for _, ni := range instance.NetworkInterfaces {
			if groupIds, exists := NetworkInterfaceHasGroup(ni, secGroup.Id); !exists {
				groupIds = append(groupIds, &secGroup.Id)
				if _, err := a.ec2Client.ModifyNetworkInterfaceAttribute(&ec2.ModifyNetworkInterfaceAttributeInput{
					NetworkInterfaceId: ni.NetworkInterfaceId,
					Groups:             groupIds,
				}); err != nil {
					log.Error(err)
				}
			}
		}
	}

	return nil
}

func (a *AWSProvider) getInstancesFromDNSNames(names types.Set[string]) ([]*ec2.Instance, error) {
	values := make([]string, len(names))
	var i int
	for k := range names {
		values[i] = k
		i++
	}

	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("private-dns-name"), Values: aws.StringSlice(values)},
		},
	}

	resp, err := a.ec2Client.DescribeInstances(input)
	if err != nil {
		return nil, err
	}

	instances := make([]*ec2.Instance, 0)
	for _, res := range resp.Reservations {
		instances = append(instances, res.Instances...)
	}

	return instances, nil
}

func New(cfg *configuration.DNS) (*AWSProvider, error) {
	meta, err := getInstanceMetaData(cfg.TagKey)
	if err != nil {
		return nil, err
	}

	sess, err := getAWSSession(meta.region)
	if err != nil {
		return nil, fmt.Errorf("aws: unable to initialise session: %s", err)
	}

	p := &AWSProvider{
		cfg: cfg,
		lookup: &cloud.LookupConfig{
			TagKey:      cfg.TagKey,
			TagValue:    meta.tagValue,
			UsePublicIP: cfg.UsePublicAddress,
		},
		ec2Client: ec2.New(sess),
		r53Client: route53.New(sess),
	}

	return p, nil
}
