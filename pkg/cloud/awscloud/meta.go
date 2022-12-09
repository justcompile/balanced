package awscloud

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

type instanceMetaData struct {
	region     string
	instanceID string
}

func getInstanceMetaData() (*instanceMetaData, error) {
	ec2meta := ec2metadata.New(session.Must(session.NewSession()))
	az, azErr := ec2meta.GetMetadata("placement/availability-zone")
	if azErr != nil {
		return nil, fmt.Errorf("awscloud: retrieving instance placement information failed: %s", azErr)
	}

	instanceId, idErr := ec2meta.GetMetadata("instance-id")
	if idErr != nil {
		return nil, fmt.Errorf("awscloud: unable to retrieve instance id: %s", idErr)
	}

	return &instanceMetaData{
		instanceID: instanceId,
		region:     az[:len(az)-1],
	}, nil
}
