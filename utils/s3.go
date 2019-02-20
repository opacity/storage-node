package utils

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/orcaman/concurrent-map"
)

type s3Wrapper struct {
	s3 *s3.S3
}

var awsPagingSize int64

var svc *s3Wrapper
var defaultBucketName string

var cachedData cmap.ConcurrentMap

func init() {
	awsPagingSize = 1000 // The max paging size per request.

	hasAwsKey := len(os.Getenv("AWS_ACCESS_KEY_ID")) > 0 && len(os.Getenv("AWS_SECRET_ACCESS_KEY")) > 0
	// Stub out the S3 if we don't have S3 access right.
	if hasAwsKey {
		svc = &s3Wrapper{
			s3: s3.New(session.Must(session.NewSession())),
		}
	} else {
		svc = &s3Wrapper{}
	}

	cachedData = cmap.New()

	defaultBucketName = "foo"
}

func isS3Enabled() bool {
	return svc.s3 != nil
}

/* Create a private bucket with bucketName. */
func createBucket(bucketName string) error {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}
	return svc.CreateBucket(input)
}

/* Delete bucket as bucketName. Must make sure no object inside the bucket*/
func deleteBucket(bucketName string) error {
	input := &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	}
	return svc.DeleteBucket(input)
}

func getObject(bucketName string, objectKey string, cached bool) (string, error) {
	if cached {
		if value, ok := cachedData.Get(getKey(bucketName, objectKey)); ok {
			return value.(string), nil
		}
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}
	data, err := svc.GetObjectAsString(input)
	if err == nil {
		cachedData.Set(getKey(bucketName, objectKey), data)
	}
	return data, err
}

func setObject(bucketName string, objectKey string, data string) error {
	input := &s3.PutObjectInput{
		Body:   aws.ReadSeekCloser(strings.NewReader(data)),
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}

	err := svc.PutObject(input)
	if err == nil {
		cachedData.Set(getKey(bucketName, objectKey), data)
	}
	return err
}

func deleteObject(bucketName string, objectKey string) error {
	cachedData.Remove(getKey(bucketName, objectKey))

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}

	return svc.DeleteObject(input)
}

func listObjectKeys(bucketName string, objectKeyPrefix string) ([]string, error) {
	var keys []string

	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucketName),
		Prefix:  aws.String(objectKeyPrefix),
		MaxKeys: aws.Int64(awsPagingSize),
	}
	err := svc.ListObjectPages(input, func(objKeys []string, lastPage bool) bool {
		keys = append(keys, objKeys...)
		return true
	})
	return keys, err
}

func deleteObjectKeys(bucketName string, objectKeyPrefix string) error {
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucketName),
		Prefix:  aws.String(objectKeyPrefix),
		MaxKeys: aws.Int64(awsPagingSize),
	}

	var deleteErr error
	err := svc.ListObjectPages(input, func(objKeys []string, lastPage bool) bool {
		var objIdentifier []*s3.ObjectIdentifier
		for _, objKey := range objKeys {
			objIdentifier = append(objIdentifier, &s3.ObjectIdentifier{Key: aws.String(objKey)})
		}
		deleteInput := &s3.DeleteObjectsInput{
			Bucket: aws.String(bucketName),
			Delete: &s3.Delete{
				Objects: objIdentifier,
			},
		}
		deleteErr = svc.DeleteObjects(deleteInput)
		if deleteErr != nil {
			return false
		}
		return true
	})

	if deleteErr != nil {
		return deleteErr
	}
	return err
}

// Get Object operation on defaultBucketName
func getDefaultBucketObject(objectKey string, cached bool) (string, error) {
	return getObject(defaultBucketName, objectKey, cached)
}

// Set Object operation on defaultBucketName
func setDefaultBucketObject(objectKey string, data string) error {
	return setObject(defaultBucketName, objectKey, data)
}

// Delete Object operation on defaultBucketName with particular prefix
func deleteDefaultBucketObject(objectKey string) error {
	return deleteObject(defaultBucketName, objectKey)
}

// List Object operation on defaultBucketName with particular prefix
func listDefaultBucketObjectKeys(objectKeyPrefix string) ([]string, error) {
	return listObjectKeys(defaultBucketName, objectKeyPrefix)
}

// Delete all the object operation on defaultBucketName with particular prefix
func deleteDefaultBucketObjectKeys(objectKeyPrefix string) error {
	return deleteObjectKeys(defaultBucketName, objectKeyPrefix)
}

func getKey(bucketName string, objectKey string) string {
	return fmt.Sprintf("%v:%v", bucketName, objectKey)
}

func (svc *s3Wrapper) CreateBucket(input *s3.CreateBucketInput) error {
	if svc.s3 == nil {
		return nil
	}

	_, err := svc.s3.CreateBucket(input)
	return err
}

func (svc *s3Wrapper) DeleteBucket(input *s3.DeleteBucketInput) error {
	if svc.s3 == nil {
		return nil
	}

	_, err := svc.s3.DeleteBucket(input)
	return err
}

func (svc *s3Wrapper) DeleteObject(input *s3.DeleteObjectInput) error {
	if svc.s3 == nil {
		return nil
	}

	_, err := svc.s3.DeleteObject(input)
	return err
}

func (svc *s3Wrapper) PutObject(input *s3.PutObjectInput) error {
	if svc.s3 == nil {
		return nil
	}

	_, err := svc.s3.PutObject(input)
	return err
}

func (svc *s3Wrapper) GetObjectAsString(input *s3.GetObjectInput) (string, error) {
	if svc.s3 == nil {
		return "", nil
	}

	output, err := svc.s3.GetObject(input)

	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(output.Body)
	return buf.String(), nil
}

func (svc *s3Wrapper) ListObjectPages(input *s3.ListObjectsV2Input, fn func([]string, bool) bool) error {
	if svc.s3 == nil {
		fn(nil, true)
		return nil
	}

	err := svc.s3.ListObjectsV2Pages(input, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		var keys []string
		for _, c := range page.Contents {
			keys = append(keys, aws.StringValue(c.Key))
		}
		return fn(keys, lastPage)
	})
	return err
}

func (svc *s3Wrapper) DeleteObjects(input *s3.DeleteObjectsInput) error {
	if svc.s3 == nil {
		return nil
	}

	_, err := svc.s3.DeleteObjects(input)
	return err
}
