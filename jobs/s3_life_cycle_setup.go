package jobs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/opacity/storage-node/utils"
)

const (
	multiUploadId    = "multi-upload"
	freeUploadId     = "free-upload"
	freeUploadPrefix = "free_upload/"
)

type s3LifeCycleSetup struct{}

func (e s3LifeCycleSetup) Run() error {
	freeUploadErr := setFreeUploadRule()
	multiUploadErr := setMultipartUploadRule()
	return utils.CollectErrors([]error{freeUploadErr, multiUploadErr})
}

func setFreeUploadRule() error {
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

func setMultipartUploadRule() error {
	rules, err := utils.GetDefaultBucketLifecycle()
	if err != nil {
		return err
	}

	for _, rule := range rules {
		if (*rule.ID) == multiUploadId {
			return nil
		}
	}

	rules = append(rules, &s3.LifecycleRule{
		AbortIncompleteMultipartUpload: &s3.AbortIncompleteMultipartUpload{
			DaysAfterInitiation: aws.Int64(1),
		},
		Status: aws.String("Enabled"),
		ID:     aws.String(multiUploadId),
		Filter: &s3.LifecycleRuleFilter{
			Prefix: aws.String(""),
		},
	})

	return utils.SetDefaultBucketLifecycle(rules)
}
