package jobs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/opacity/storage-node/utils"
)

const (
	freeUploadId     = "free-upload"
	freeUploadPrefix = "free_upload/"
)

type s3LifeCycleSetup struct{}

func (e s3LifeCycleSetup) Run() error {
	rules, err := utils.GetDefaultBucketLifecycle()
	if err != nil {
		return err
	}

	for _, rule := range rules {
		if (*rule.ID) == freeUploadId {
			return nil
		}
	}

	rules = append(rules, &s3.LifecycleRule{
		Expiration: &s3.LifecycleExpiration{
			Days: aws.Int64(30),
		},
		Filter: &s3.LifecycleRuleFilter{
			Prefix: aws.String(freeUploadPrefix),
		},
		ID:     aws.String(freeUploadId),
		Status: aws.String("Enabled"),
	})

	return utils.SetDefaultBucketLifecycle(rules)
}
