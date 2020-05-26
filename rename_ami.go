package main

import (
	"log"
	"regexp"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go/service/sms"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Trim the last character of the string passed in.
func trimLastChar(s string) string {
	r, size := utf8.DecodeLastRuneInString(s)
	if r == utf8.RuneError && (size == 0 || size == 1) {
		size = 0
	}
	return s[:len(s)-size]
}

// GetAccountID will return the AWS Account ID
func GetAccountID(sess *session.Session) string {
	// Create STS Client to get Account ID
	stsSvc := sts.New(sess)
	accountID, err := stsSvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		log.Fatal(err)
	}
	return *accountID.Account
}

func main() {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})

	if err != nil {
		log.Fatal(err)
	}

	accountID := GetAccountID(sess)

	// Create EC2 client
	ec2Svc := ec2.New(sess)
	input := &ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("owner-id"),
				Values: []*string{aws.String(accountID)},
			},
		},
	}
	resp, err := ec2Svc.DescribeImages(input)

	if err != nil {
		log.Fatal(err)
	}

	smsSvc := sms.New(sess)
	// Iterate through each image, and locate the VM Name
	for _, image := range resp.Images {
		re := regexp.MustCompile(`sms-job.+?(/)`)
		smsJobID := trimLastChar(re.FindString(*image.ImageLocation))

		input := &sms.GetReplicationJobsInput{
			ReplicationJobId: aws.String(smsJobID),
		}
		vmName, err := smsSvc.GetReplicationJobs(input)
		if err != nil {
			log.Fatal(err)
		}

		_, err = ec2Svc.CreateTags(
			&ec2.CreateTagsInput{
				Resources: []*string{
					aws.String(*image.ImageId),
				},
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(*vmName.ReplicationJobList[0].VmServer.VmName),
					},
				},
			},
		)

		if err != nil {
			log.Fatal(err)
		}

	}

}
