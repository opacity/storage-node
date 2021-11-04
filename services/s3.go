package services

import (
	"bytes"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/meirf/gopart"
)

type ObjectIterator func([]*s3.Object) bool

type S3Wrapper struct {
	S3                  *s3.S3
	ObjectIterator      ObjectIterator
	MaxMultiPartRetries int
	AwsPagingSize       int64
}

func (svc *S3Wrapper) CreateBucket(bucketName string) error {
	if svc.S3 == nil {
		return nil
	}

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := svc.S3.CreateBucket(input)
	return err
}

func (svc *S3Wrapper) DeleteBucket(bucketName string) error {
	if svc.S3 == nil {
		return nil
	}

	input := &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := svc.S3.DeleteBucket(input)
	return err
}

func (svc *S3Wrapper) DoesObjectExist(bucketName string, objectKey string) bool {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}

	r, err := svc.S3.HeadObject(input)

	return err == nil && r != nil
}

func (svc *S3Wrapper) GetObjectSizeInByte(bucketName string, objectKey string) int64 {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}

	r, err := svc.S3.HeadObject(input)
	if err != nil {
		return 0
	}
	return aws.Int64Value(r.ContentLength)
}

func (svc *S3Wrapper) GetObject(bucketName, objectKey, downloadRange string) (string, error) {

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}
	if downloadRange != "" {
		input.SetRange(downloadRange)
	}

	output, err := svc.S3.GetObject(input)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(output.Body)
	outputString := buf.String()

	return outputString, err
}

func (svc *S3Wrapper) GetObjectOutput(bucketName, objectKey, downloadRange string) (*s3.GetObjectOutput, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}
	if downloadRange != "" {
		input.SetRange(downloadRange)
	}
	output, err := svc.S3.GetObject(input)

	return output, err
}

// @TODO: remove this?
func (svc *S3Wrapper) GetObjectAsString(bucketName, objectKey, downloadRange string) (string, error) {
	outputString, err := svc.GetObject(bucketName, objectKey, downloadRange)
	if err != nil {
		return "", err
	}

	return outputString, nil
}

func (svc *S3Wrapper) SetObject(bucketName, objectKey, data, fileContentType string) error {
	input := &s3.PutObjectInput{
		Body:        aws.ReadSeekCloser(strings.NewReader(data)),
		Bucket:      aws.String(bucketName),
		Key:         aws.String(objectKey),
		ContentType: aws.String(fileContentType),
	}

	err := svc.PutObject(input)

	return err
}

func (svc *S3Wrapper) DeleteObject(bucketName string, objectKey string) error {
	if svc.S3 == nil {
		return nil
	}

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}
	_, err := svc.S3.DeleteObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.RequestFailure); ok {
			if aerr.StatusCode() == 404 {
				return nil
			}
		}
	}
	return err
}

func (svc *S3Wrapper) PutObject(input *s3.PutObjectInput) error {
	if svc.S3 == nil {
		return nil
	}

	_, err := svc.S3.PutObject(input)
	return err
}

func (svc *S3Wrapper) ListObjectKeys(bucketName string, objectKeyPrefix string) ([]string, error) {
	var keys []string

	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucketName),
		Prefix:  aws.String(objectKeyPrefix),
		MaxKeys: aws.Int64(svc.AwsPagingSize),
	}

	err := svc.ListObjectPages(input, func(objs []*s3.Object) bool {
		for _, c := range objs {
			keys = append(keys, aws.StringValue(c.Key))
		}
		return true
	})
	return keys, err
}

func (svc *S3Wrapper) IterateBucketAllObjects(bucketName string, i ObjectIterator) error {
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucketName),
		MaxKeys: aws.Int64(svc.AwsPagingSize),
	}
	return svc.ListObjectPages(input, i)
}

func (svc *S3Wrapper) ListObjectPages(input *s3.ListObjectsV2Input, it ObjectIterator) error {
	if svc.S3 == nil {
		it(nil)
		return nil
	}

	err := svc.S3.ListObjectsV2Pages(input, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		return it(page.Contents)
	})
	return err
}

func (svc *S3Wrapper) DeleteObjectKeys(bucketName string, objectKeyPrefix string) error {
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucketName),
		Prefix:  aws.String(objectKeyPrefix),
		MaxKeys: aws.Int64(svc.AwsPagingSize),
	}

	var deleteErr error

	err := svc.ListObjectPages(input, func(objs []*s3.Object) bool {
		var objIdentifier []*s3.ObjectIdentifier
		for _, c := range objs {
			objIdentifier = append(objIdentifier, &s3.ObjectIdentifier{Key: c.Key})
		}
		deleteInput := &s3.DeleteObjectsInput{
			Bucket: aws.String(bucketName),
			Delete: &s3.Delete{
				Objects: objIdentifier,
			},
		}
		_, deleteErr = svc.S3.DeleteObjects(deleteInput)

		return deleteErr == nil
	})

	if deleteErr != nil {
		return deleteErr
	}
	return err
}

func (svc *S3Wrapper) DeleteObjects(bucketName string, objectKeys []string) error {
	if svc.S3 == nil {
		return nil
	}

	input := &s3.DeleteObjectsInput{
		Bucket: aws.String(bucketName),
	}
	for idRange := range gopart.Partition(len(objectKeys), int(svc.AwsPagingSize)) {
		var objIdentifier []*s3.ObjectIdentifier
		for _, c := range objectKeys[idRange.Low:idRange.High] {
			objIdentifier = append(objIdentifier, &s3.ObjectIdentifier{Key: aws.String(c)})
		}
		input.Delete = &s3.Delete{
			Objects: objIdentifier,
			Quiet:   aws.Bool(true),
		}

		if _, err := svc.S3.DeleteObjects(input); err != nil {
			return err
		}
	}
	return nil
}

func (svc *S3Wrapper) HeadObject(input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	if svc.S3 == nil {
		return nil, nil
	}

	out, err := svc.S3.HeadObject(input)
	return out, err
}

func (svc *S3Wrapper) StartMultipartUpload(bucketName, key, fileType string) (*string, *string, error) {
	if svc.S3 == nil {
		return nil, nil, nil
	}

	input := &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(key),
		ContentType: aws.String(fileType),
	}

	output, err := svc.S3.CreateMultipartUpload(input)

	return output.Key, output.UploadId, err
}

func (svc *S3Wrapper) UploadPartOfMultiPartUpload(bucketName, key, uploadID string, fileBytes []byte,
	partNumber int) (*s3.CompletedPart, error) {
	tryNum := 1
	partInput := &s3.UploadPartInput{
		Body:          bytes.NewReader(fileBytes),
		Bucket:        aws.String(bucketName),
		Key:           aws.String(key),
		UploadId:      aws.String(uploadID),
		PartNumber:    aws.Int64(int64(partNumber)),
		ContentLength: aws.Int64(int64(len(fileBytes))),
	}

	for tryNum <= svc.MaxMultiPartRetries {
		uploadResult, err := svc.S3.UploadPart(partInput)
		if err != nil {
			if tryNum == svc.MaxMultiPartRetries {
				if aerr, ok := err.(awserr.Error); ok {
					return nil, aerr
				}
				return nil, err
			}
			tryNum++
		} else {
			return &s3.CompletedPart{
				ETag:       uploadResult.ETag,
				PartNumber: aws.Int64(int64(partNumber)),
			}, nil
		}
	}
	return nil, nil
}

func (svc *S3Wrapper) FinishMultipartUpload(bucketName, key, uploadID string,
	completedParts []*s3.CompletedPart) (*s3.CompleteMultipartUploadOutput, error) {
	completeInput := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucketName),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completedParts,
		},
	}
	return svc.S3.CompleteMultipartUpload(completeInput)
}

func (svc *S3Wrapper) CancelMultipartUpload(bucketName, key, uploadID string) error {
	abortInput := &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(bucketName),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	}
	_, err := svc.S3.AbortMultipartUpload(abortInput)
	return err
}

func (svc *S3Wrapper) SetObjectCannedAcl(bucketName string, objectName string, cannedAcl string) error {
	if svc.S3 == nil {
		return nil
	}

	input := &s3.PutObjectAclInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectName),
		ACL:    aws.String(cannedAcl),
	}

	_, err := svc.S3.PutObjectAcl(input)
	return err
}

func (svc *S3Wrapper) PutBucketLifecycleConfiguration(bucketName string, rules []*s3.LifecycleRule) error {
	if svc.S3 == nil {
		return nil
	}
	input := &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucketName),
		LifecycleConfiguration: &s3.BucketLifecycleConfiguration{
			Rules: rules,
		},
	}
	_, err := svc.S3.PutBucketLifecycleConfiguration(input)
	return err
}

func (svc *S3Wrapper) GetBucketLifecycleConfiguration(bucketName string) ([]*s3.LifecycleRule, error) {
	if svc.S3 == nil {
		return nil, nil
	}

	input := &s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucketName),
	}

	v, err := svc.S3.GetBucketLifecycleConfiguration(input)
	if err != nil {
		if aerr, ok := err.(awserr.RequestFailure); ok {
			if aerr.StatusCode() == 404 {
				return nil, nil
			}
		}
		return nil, err
	}
	return v.Rules, nil
}

// @TODO: Complete this
func (svc *S3Wrapper) DownloadS3ObjectInChunks(key, downloadRange string) {

}
