package awscloud

import (
	"balanced/pkg/types"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	log "github.com/sirupsen/logrus"
)

func NetworkInterfaceHasGroup(ini *ec2.InstanceNetworkInterface, groupId string) ([]string, bool) {
	hasGroup := false

	groupIds := make([]string, len(ini.Groups))
	for i, grp := range ini.Groups {
		groupIds[i] = *grp.GroupId
		if *grp.GroupId == groupId {
			hasGroup = true
		}
	}
	log.Debugf("network interface %s has groups: %v, (checking for group %s)", *ini.NetworkInterfaceId, groupIds, groupId)

	return groupIds, hasGroup
}

func ipPermissionsFromPorts(ports types.Set[int64], destinationGroupId, vpcId *string) []*ec2.IpPermission {
	perms := make([]*ec2.IpPermission, len(ports))
	var i int

	for p := range ports {
		perms[i] = (&ec2.IpPermission{}).
			SetFromPort(p).
			SetToPort(p).
			SetIpProtocol(ec2.ProtocolTcp).
			SetUserIdGroupPairs([]*ec2.UserIdGroupPair{
				{GroupId: destinationGroupId, VpcId: vpcId, Description: aws.String("Managed by balanced")},
			})
	}

	return perms
}
