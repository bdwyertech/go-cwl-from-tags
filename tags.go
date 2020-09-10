// Encoding: UTF-8

package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func getTags() (tags []*ec2.TagDescription) {
	// AWS Session
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config:            *aws.NewConfig().WithCredentialsChainVerboseErrors(true),
		SharedConfigState: session.SharedConfigDisable,
	}))

	metadata := ec2metadata.New(sess)

	if !metadata.Available() {
		log.Fatal("EC2 Metadata is not available... Are we running on an EC2 instance?")
	}

	identity, err := metadata.GetInstanceIdentityDocument()
	if err != nil {
		log.Fatal(err)
	}
	instanceID := identity.InstanceID
	sess.Config = sess.Config.WithRegion(identity.Region)

	ec2client := ec2.New(sess)

	input := &ec2.DescribeTagsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("resource-id"),
				Values: []*string{
					aws.String(instanceID),
				},
			},
		},
	}

	resp, err := ec2client.DescribeTags(input)
	if err != nil {
		log.Fatal(err)
	}

	tags = resp.Tags

	// Handle EC2 API Pagination
	for {
		if resp.NextToken == nil {
			break
		}

		input.NextToken = resp.NextToken

		resp, err := ec2client.DescribeTags(input)
		if err != nil {
			log.Fatal(err)
		}

		tags = append(tags, resp.Tags...)
	}

	return
}
