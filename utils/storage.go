package utils

import (
	"fmt"

	"github.com/opacity/storage-node/services"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	MaxMultiPartSize          = int64(1024 * 1024 * 50)
	MinMultiPartSize          = int64(1024 * 1024 * 5)
	MaxMultiPartRetries       = 10
	AwsPagingSize             = 1000
	CannedAcl_Private         = "private"
	CannedAcl_PublicRead      = "public-read"
	CannedAcl_PublicReadWrite = "public-read-write"
	DefaultFileContentType    = "application/octet-stream"
)

var s3Svc *services.S3Wrapper
var minIoSvc *services.S3Wrapper

func newStorageSession() {
	s3Svc = &services.S3Wrapper{
		S3: s3.New(session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))),
		MaxMultiPartRetries: MaxMultiPartRetries,
		AwsPagingSize:       AwsPagingSize,
	}

	minIoConfig := &aws.Config{
		Credentials: credentials.NewStaticCredentials(Env.GuardianAccessKeyID, Env.GuardianSecretAccessKey, ""),
		Endpoint:    aws.String(Env.GuardianEndpoint),
		// @TODO: Is Region needed?
		Region:           aws.String("us-east-1"),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
	}
	newMinIOSession, newMinIOSessionErr := session.NewSession(minIoConfig)

	minIoSvc = &services.S3Wrapper{
		S3:                  s3.New(session.Must(newMinIOSession, newMinIOSessionErr)),
		MaxMultiPartRetries: MaxMultiPartRetries,
		AwsPagingSize:       AwsPagingSize,
	}
}

func IsS3Enabled() bool {
	return s3Svc.S3 != nil
}

func IsMinIoEnable() bool {
	return minIoSvc.S3 != nil
}

func DoesDefaultBucketObjectExist(objectKey string, storageType FileStorageType) bool {
	if storageType == Galaxy {
		return minIoSvc.DoesObjectExist(Env.GuardianBucketName, objectKey)
	}
	return s3Svc.DoesObjectExist(Env.BucketName, objectKey)
}

// Get Object operation on defaultBucketName
func GetDefaultBucketObject(objectKey string, storageType FileStorageType) (string, error) {
	if storageType == Galaxy {
		return minIoSvc.GetObjectAsString(Env.GuardianBucketName, objectKey, "")
	}
	return s3Svc.GetObjectAsString(Env.BucketName, objectKey, "")
}

func GetBucketObject(objectKey, downloadRange string, storageType FileStorageType) (*s3.GetObjectOutput, error) {
	if storageType == Galaxy {
		return minIoSvc.GetObjectOutput(Env.GuardianBucketName, objectKey, downloadRange)
	}
	return s3Svc.GetObjectOutput(Env.BucketName, objectKey, downloadRange)
}

func GetDefaultBucketObjectSize(objectKey string, storageType FileStorageType) int64 {
	if storageType == Galaxy {
		return minIoSvc.GetObjectSizeInByte(Env.GuardianBucketName, objectKey)
	}

	return s3Svc.GetObjectSizeInByte(Env.BucketName, objectKey)
}

// Set Object operation on defaultBucketName
func SetDefaultBucketObject(objectKey, data, fileContentType string, storageType FileStorageType) error {
	if fileContentType == "" {
		fileContentType = DefaultFileContentType
	}
	if storageType == Galaxy {
		return minIoSvc.SetObject(Env.GuardianBucketName, objectKey, data, fileContentType)
	}

	return s3Svc.SetObject(Env.BucketName, objectKey, data, fileContentType)
}

// Delete Object operation on defaultBucketName with particular prefix
func DeleteDefaultBucketObject(objectKey string, storageType FileStorageType) error {
	if storageType == Galaxy {
		return minIoSvc.DeleteObject(Env.GuardianBucketName, objectKey)
	}
	return s3Svc.DeleteObject(Env.BucketName, objectKey)
}

// List Object operation on defaultBucketName with particular prefix
func ListDefaultBucketObjectKeys(objectKeyPrefix string, storageType FileStorageType) ([]string, error) {
	if storageType == Galaxy {
		return minIoSvc.ListObjectKeys(Env.GuardianBucketName, objectKeyPrefix)
	}
	return s3Svc.ListObjectKeys(Env.BucketName, objectKeyPrefix)
}

// Delete all the object operation on defaultBucketName with particular prefix
func DeleteDefaultBucketObjectKeys(objectKeyPrefix string, storageType FileStorageType) error {
	if storageType == Galaxy {
		return minIoSvc.DeleteObjectKeys(Env.GuardianBucketName, objectKeyPrefix)
	}
	return s3Svc.DeleteObjectKeys(Env.BucketName, objectKeyPrefix)
}

func CreateMultiPartUpload(key, fileContentType string, storageType FileStorageType) (*string, *string, error) {
	if fileContentType == "" {
		fileContentType = DefaultFileContentType
	}

	if storageType == Galaxy {
		return minIoSvc.StartMultipartUpload(Env.GuardianBucketName, key, fileContentType)
	}

	return s3Svc.StartMultipartUpload(Env.BucketName, key, fileContentType)
}

func UploadMultiPartPart(key, uploadID string, fileBytes []byte, partNumber int, storageType FileStorageType) (*s3.CompletedPart, error) {
	if storageType == Galaxy {
		return minIoSvc.UploadPartOfMultiPartUpload(Env.GuardianBucketName, key, uploadID, fileBytes, partNumber)
	}

	return s3Svc.UploadPartOfMultiPartUpload(Env.BucketName, key, uploadID, fileBytes, partNumber)
}

func CompleteMultiPartUpload(key, uploadID string,
	completedParts []*s3.CompletedPart, storageType FileStorageType) (*s3.CompleteMultipartUploadOutput, error) {
	if storageType == Galaxy {
		return minIoSvc.FinishMultipartUpload(Env.GuardianBucketName, key, uploadID, completedParts)
	}

	return s3Svc.FinishMultipartUpload(Env.BucketName, key, uploadID, completedParts)
}

func AbortMultiPartUpload(key, uploadID string, storageType FileStorageType) error {
	if storageType == Galaxy {
		return minIoSvc.CancelMultipartUpload(Env.GuardianBucketName, key, uploadID)
	}

	return s3Svc.CancelMultipartUpload(Env.BucketName, key, uploadID)
}
func SetDefaultObjectCannedAcl(objectKey string, cannedAcl string, storageType FileStorageType) error {
	if storageType == Galaxy {
		// minIoSvc.SetObjectCannedAcl(Env.GuardianBucketName, objectKey, cannedAcl)
		return nil
	}

	return s3Svc.SetObjectCannedAcl(Env.BucketName, objectKey, cannedAcl)
}

func SetDefaultBucketLifecycle(rules []*s3.LifecycleRule, storageType FileStorageType) error {
	if storageType == Galaxy {
		return minIoSvc.PutBucketLifecycleConfiguration(Env.GuardianBucketName, rules)
	}

	return s3Svc.PutBucketLifecycleConfiguration(Env.BucketName, rules)
}

func GetDefaultBucketLifecycle(storageType FileStorageType) ([]*s3.LifecycleRule, error) {
	if storageType == Galaxy {
		return minIoSvc.GetBucketLifecycleConfiguration(Env.GuardianBucketName)
	}

	return s3Svc.GetBucketLifecycleConfiguration(Env.BucketName)
}

func IterateDefaultBucketAllObjects(i services.ObjectIterator, storageType FileStorageType) error {
	if storageType == Galaxy {
		return minIoSvc.IterateBucketAllObjects(Env.GuardianBucketName, i)
	}

	return s3Svc.IterateBucketAllObjects(Env.BucketName, i)
}

func DeleteDefaultBucketObjects(objectKeys []string, storageType FileStorageType) error {
	if storageType == Galaxy {
		return minIoSvc.DeleteObjects(Env.GuardianBucketName, objectKeys)
	}

	return s3Svc.DeleteObjects(Env.BucketName, objectKeys)
}

func GetStorageURL(storageType FileStorageType) string {
	if storageType == Galaxy {
		return fmt.Sprintf("%s/%s/", Env.GuardianEndpoint, Env.GuardianBucketName)
	}
	return fmt.Sprintf("https://s3.%s.amazonaws.com/%s/", Env.AwsRegion, Env.BucketName)
}
