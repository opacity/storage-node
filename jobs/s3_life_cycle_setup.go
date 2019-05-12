package jobs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/opacity/storage-node/utils"
)

const (
	multiUploadId = "multi-upload"

	freeUploadId     = "free-upload"
	freeUploadPrefix = "free_upload/"

	testPrefixId = "unit-test"
	testPrefix   = "test/"
)

type s3LifeCycleSetup struct{}

func (e s3LifeCycleSetup) Run() error {
	lifecycles := getLifecyclesMap()

	rules, err := utils.GetDefaultBucketLifecycle()
	if err != nil {
		return err
	}

	for k, v := range lifecycles {
		hasKey := false
		for _, rule := range rules {
			if (*rule.ID) == k {
				hasKey = true
				break
			}
		}
		if !hasKey {
			rules = append(rules, &v)
		}
	}
	return utils.SetDefaultBucketLifecycle(rules)
}

func getLifecyclesMap() map[string]s3.LifecycleRule {
	return map[string]s3.LifecycleRule{
		freeUploadId: s3.LifecycleRule{
			Expiration: &s3.LifecycleExpiration{
				Days: aws.Int64(30),
			},
			Filter: &s3.LifecycleRuleFilter{
				Prefix: aws.String(freeUploadPrefix),
			},
			ID:     aws.String(freeUploadId),
			Status: aws.String("Enabled"),
		},
		multiUploadId: s3.LifecycleRule{
			AbortIncompleteMultipartUpload: &s3.AbortIncompleteMultipartUpload{
				DaysAfterInitiation: aws.Int64(1),
			},
			Status: aws.String("Enabled"),
			ID:     aws.String(multiUploadId),
			Filter: &s3.LifecycleRuleFilter{
				Prefix: aws.String(""),
			},
		},
		testPrefixId: s3.LifecycleRule{
			Expiration: &s3.LifecycleExpiration{
				Days: aws.Int64(1),
			},
			Filter: &s3.LifecycleRuleFilter{
				Prefix: aws.String(testPrefix),
			},
			ID:     aws.String(testPrefixId),
			Status: aws.String("Enabled"),
		},
	}
}
