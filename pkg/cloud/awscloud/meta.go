package awscloud

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

type instanceMetaData struct {
	region   string
	tagValue string
}

func getInstanceMetaData(tagKey string) (*instanceMetaData, error) {
	ec2meta := ec2metadata.New(session.Must(session.NewSession()))
	az, azErr := ec2meta.GetMetadata("placement/availability-zone")
	if azErr != nil {
		return nil, fmt.Errorf("awscloud: retrieving instance placement information failed: %s", azErr)
	}

	tagValue, tagErr := ec2meta.GetMetadata("tags/instance/" + tagKey)
	if tagErr != nil {
		return nil, fmt.Errorf("awscloud: retrieving instance tagging information failed: %s", tagErr)
	}

	return &instanceMetaData{
		region:   az[:len(az)-1],
		tagValue: tagValue,
	}, nil
}
