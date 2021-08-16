package utils

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/meirf/gopart"
	cmap "github.com/orcaman/concurrent-map"
)

type s3Wrapper struct {
	s3 *s3.S3
}

type ObjectIterator func([]*s3.Object) bool

const (
	MaxMultiPartSize          = int64(1024 * 1024 * 50)
	MinMultiPartSize          = int64(1024 * 1024 * 5)
	MaxMultiPartRetries       = 10
	CannedAcl_Private         = "private"
	CannedAcl_PublicRead      = "public-read"
	CannedAcl_PublicReadWrite = "public-read-write"
	DefaultFileContentType    = "application/octet-stream"
)

var awsPagingSize int64
var svc *s3Wrapper
var cachedData cmap.ConcurrentMap
var shouldCachedData bool

func init() {
	awsPagingSize = 1000 // The max paging size per request.
	shouldCachedData = false
	cachedData = cmap.New()
}

func newS3Session() {
	svc = &s3Wrapper{
		s3: s3.New(session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))),
	}
}

func SetS3DataCaching(isCaching bool) {
	shouldCachedData = isCaching
}

func IsS3Enabled() bool {
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

func doesObjectExist(bucketName string, objectKey string) bool {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}

	r, err := svc.HeadObject(input)
	return err == nil && r != nil
}

func getObjectSizeInByte(bucketName string, objectKey string) int64 {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}

	r, err := svc.HeadObject(input)
	if err != nil {
		return 0
	}
	return aws.Int64Value(r.ContentLength)
}

func getObject(bucketName, objectKey, downloadRange string, cached bool) (string, error) {
	if cached {
		valueS, okS := cachedData.Get(getKey(bucketName, objectKey))
		if okS {
			return valueS.(string), nil
		}
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}
	if downloadRange != "" {
		input.SetRange(downloadRange)
	}

	output, err := svc.s3.GetObject(input)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(output.Body)
	outputString := buf.String()

	if err == nil && shouldCachedData {
		cachedData.Set(getKey(bucketName, objectKey), outputString)
	}

	return outputString, err
}

func getObjectOutput(bucketName, objectKey, downloadRange string, cached bool) (*s3.GetObjectOutput, error) {
	if cached {
		valueR, okR := cachedData.Get(getKey(bucketName, objectKey+"_object"))
		if okR {
			return valueR.(*s3.GetObjectOutput), nil
		}
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}
	if downloadRange != "" {
		input.SetRange(downloadRange)
	}
	output, err := svc.s3.GetObject(input)

	if err == nil && shouldCachedData {
		cachedData.Set(getKey(bucketName, objectKey+"_object"), output)
	}

	return output, err
}

func getObjectAsString(bucketName, objectKey, downloadRange string, cached bool) (string, error) {
	outputString, err := getObject(bucketName, objectKey, downloadRange, cached)
	if err != nil {
		return "", err
	}

	return outputString, nil
}

func setObject(bucketName, objectKey, data, fileContentType string) error {
	input := &s3.PutObjectInput{
		Body:        aws.ReadSeekCloser(strings.NewReader(data)),
		Bucket:      aws.String(bucketName),
		Key:         aws.String(objectKey),
		ContentType: aws.String(fileContentType),
	}

	err := svc.PutObject(input)
	if err == nil && shouldCachedData {
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

	err := svc.ListObjectPages(input, func(objs []*s3.Object) bool {
		for _, c := range objs {
			keys = append(keys, aws.StringValue(c.Key))
		}
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
		deleteErr = svc.DeleteObjects(deleteInput)
		return deleteErr == nil
	})

	if deleteErr != nil {
		return deleteErr
	}
	return err
}

func createMultiPartUpload(key, fileType string) (*string, *string, error) {
	output, err := svc.StartMultipartUpload(key, fileType)

	return output.Key, output.UploadId, err
}

func uploadPart(key, uploadID string, fileBytes []byte, partNumber int) (*s3.CompletedPart, error) {
	return svc.UploadPartOfMultiPartUpload(key, uploadID, fileBytes, partNumber)
}

func abortMultiPartUpload(key, uploadID string) error {
	return svc.CancelMultipartUpload(key, uploadID)
}

func completeMultiPartUpload(key, uploadID string,
	completedParts []*s3.CompletedPart) (*s3.CompleteMultipartUploadOutput, error) {
	return svc.FinishMultipartUpload(key, uploadID, completedParts)
}

func setObjectCannedAcl(bucketName string, objectName string, cannedAcl string) error {
	input := &s3.PutObjectAclInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectName),
		ACL:    aws.String(cannedAcl),
	}

	return svc.SetObjectCannedAcl(input)
}

func setBucketLifecycle(bucketName string, rules []*s3.LifecycleRule) error {
	input := &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucketName),
		LifecycleConfiguration: &s3.BucketLifecycleConfiguration{
			Rules: rules,
		},
	}

	return svc.PutBucketLifecycleConfiguration(input)
}

func getBucketLifecycle(bucketName string) ([]*s3.LifecycleRule, error) {
	input := &s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucketName),
	}
	return svc.GetBucketLifecycleConfiguration(input)
}

func iterateBucketAllObjects(bucketName string, i ObjectIterator) error {
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucketName),
		MaxKeys: aws.Int64(awsPagingSize),
	}
	return svc.ListObjectPages(input, i)
}

func deleteObjects(bucketName string, objectKeys []string) error {
	input := &s3.DeleteObjectsInput{
		Bucket: aws.String(bucketName),
	}

	for idRange := range gopart.Partition(len(objectKeys), int(awsPagingSize)) {
		var objIdentifier []*s3.ObjectIdentifier
		for _, c := range objectKeys[idRange.Low:idRange.High] {
			objIdentifier = append(objIdentifier, &s3.ObjectIdentifier{Key: aws.String(c)})
		}
		input.Delete = &s3.Delete{
			Objects: objIdentifier,
			Quiet:   aws.Bool(true),
		}

		if err := svc.DeleteObjects(input); err != nil {
			return err
		}
	}
	return nil
}

func DoesDefaultBucketObjectExist(objectKey string) bool {
	return doesObjectExist(Env.BucketName, objectKey)
}

// Get Object operation on defaultBucketName
func GetDefaultBucketObject(objectKey string, cached bool) (string, error) {
	return getObjectAsString(Env.BucketName, objectKey, "", cached)
}

func GetBucketObject(objectKey, downloadRange string, cached bool) (*s3.GetObjectOutput, error) {
	return getObjectOutput(Env.BucketName, objectKey, downloadRange, cached)
}

func GetDefaultBucketObjectSize(objectKey string) int64 {
	return getObjectSizeInByte(Env.BucketName, objectKey)
}

// Set Object operation on defaultBucketName
func SetDefaultBucketObject(objectKey, data, fileContentType string) error {
	if fileContentType == "" {
		fileContentType = DefaultFileContentType
	}
	return setObject(Env.BucketName, objectKey, data, fileContentType)
}

// Delete Object operation on defaultBucketName with particular prefix
func DeleteDefaultBucketObject(objectKey string) error {
	return deleteObject(Env.BucketName, objectKey)
}

// List Object operation on defaultBucketName with particular prefix
func ListDefaultBucketObjectKeys(objectKeyPrefix string) ([]string, error) {
	return listObjectKeys(Env.BucketName, objectKeyPrefix)
}

// Delete all the object operation on defaultBucketName with particular prefix
func DeleteDefaultBucketObjectKeys(objectKeyPrefix string) error {
	return deleteObjectKeys(Env.BucketName, objectKeyPrefix)
}

func CreateMultiPartUpload(key, fileContentType string) (*string, *string, error) {
	if fileContentType == "" {
		fileContentType = DefaultFileContentType
	}
	return createMultiPartUpload(key, fileContentType)
}

func UploadMultiPartPart(key, uploadID string, fileBytes []byte, partNumber int) (*s3.CompletedPart, error) {
	return uploadPart(key, uploadID, fileBytes, partNumber)
}

func AbortMultiPartUpload(key, uploadID string) error {
	return abortMultiPartUpload(key, uploadID)
}

func CompleteMultiPartUpload(key, uploadID string,
	completedParts []*s3.CompletedPart) (*s3.CompleteMultipartUploadOutput, error) {
	return completeMultiPartUpload(key, uploadID, completedParts)
}

func SetDefaultObjectCannedAcl(objectKey string, cannedAcl string) error {
	return setObjectCannedAcl(Env.BucketName, objectKey, cannedAcl)
}

func SetDefaultBucketLifecycle(rules []*s3.LifecycleRule) error {
	return setBucketLifecycle(Env.BucketName, rules)
}

func GetDefaultBucketLifecycle() ([]*s3.LifecycleRule, error) {
	return getBucketLifecycle(Env.BucketName)
}

func IterateDefaultBucketAllObjects(i ObjectIterator) error {
	return iterateBucketAllObjects(Env.BucketName, i)
}

func DeleteDefaultBucketObjects(objectKeys []string) error {
	return deleteObjects(Env.BucketName, objectKeys)
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
	if err != nil {
		if aerr, ok := err.(awserr.RequestFailure); ok {
			if aerr.StatusCode() == 404 {
				return nil
			}
		}
	}
	return err
}

func (svc *s3Wrapper) PutObject(input *s3.PutObjectInput) error {
	if svc.s3 == nil {
		return nil
	}

	_, err := svc.s3.PutObject(input)
	return err
}

func (svc *s3Wrapper) ListObjectPages(input *s3.ListObjectsV2Input, it ObjectIterator) error {
	if svc.s3 == nil {
		it(nil)
		return nil
	}

	err := svc.s3.ListObjectsV2Pages(input, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		return it(page.Contents)
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

func (svc *s3Wrapper) HeadObject(input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	if svc.s3 == nil {
		return nil, nil
	}

	out, err := svc.s3.HeadObject(input)
	return out, err
}

func (svc *s3Wrapper) SetObjectCannedAcl(input *s3.PutObjectAclInput) error {
	if svc.s3 == nil {
		return nil
	}

	_, err := svc.s3.PutObjectAcl(input)
	return err
}

func (svc *s3Wrapper) StartMultipartUpload(key, fileType string) (*s3.CreateMultipartUploadOutput, error) {
	if svc.s3 == nil {
		return &s3.CreateMultipartUploadOutput{}, nil
	}

	input := &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(Env.BucketName),
		Key:         aws.String(key),
		ContentType: aws.String(fileType),
	}

	return svc.s3.CreateMultipartUpload(input)
}

func (svc *s3Wrapper) UploadPartOfMultiPartUpload(key, uploadID string, fileBytes []byte,
	partNumber int) (*s3.CompletedPart, error) {
	tryNum := 1
	partInput := &s3.UploadPartInput{
		Body:          bytes.NewReader(fileBytes),
		Bucket:        aws.String(Env.BucketName),
		Key:           aws.String(key),
		UploadId:      aws.String(uploadID),
		PartNumber:    aws.Int64(int64(partNumber)),
		ContentLength: aws.Int64(int64(len(fileBytes))),
	}

	for tryNum <= MaxMultiPartRetries {
		uploadResult, err := svc.s3.UploadPart(partInput)
		if err != nil {
			if tryNum == MaxMultiPartRetries {
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

func (svc *s3Wrapper) FinishMultipartUpload(key, uploadID string,
	completedParts []*s3.CompletedPart) (*s3.CompleteMultipartUploadOutput, error) {
	completeInput := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(Env.BucketName),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completedParts,
		},
	}
	return svc.s3.CompleteMultipartUpload(completeInput)
}

func (svc *s3Wrapper) CancelMultipartUpload(key, uploadID string) error {
	abortInput := &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(Env.BucketName),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	}
	_, err := svc.s3.AbortMultipartUpload(abortInput)
	return err
}

func (svc *s3Wrapper) PutBucketLifecycleConfiguration(input *s3.PutBucketLifecycleConfigurationInput) error {
	if svc.s3 == nil {
		return nil
	}

	_, err := svc.s3.PutBucketLifecycleConfiguration(input)
	return err
}

func (svc *s3Wrapper) GetBucketLifecycleConfiguration(input *s3.GetBucketLifecycleConfigurationInput) ([]*s3.LifecycleRule, error) {
	if svc.s3 == nil {
		return nil, nil
	}

	v, err := svc.s3.GetBucketLifecycleConfiguration(input)
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

func (svc *s3Wrapper) DownloadS3ObjectInChunks(key, downloadRange string) {

}
