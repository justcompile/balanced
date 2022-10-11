package discovery

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func AWSAddrs(cfg *LookupConfig) ([]string, error) {

	config := aws.Config{
		// Region: &region,
		Credentials: credentials.NewChainCredentials(
			[]credentials.Provider{
				&credentials.EnvProvider{},
				&credentials.SharedCredentialsProvider{},
				defaults.RemoteCredProvider(*(defaults.Config()), defaults.Handlers()),
			},
		),
	}

	sess, err := session.NewSession(&config)
	if err != nil {
		return nil, fmt.Errorf("unable to create aws session: %s", err)
	}

	svc := ec2.New(sess)

	resp, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("tag:" + cfg.TagKey),
				Values: []*string{aws.String(cfg.TagValue)},
			},
			&ec2.Filter{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String("running")},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("discover-aws: DescribeInstancesInput failed: %s", err)
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
