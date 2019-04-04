package utils

import (
	"os"
	"testing"

	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/assert"
)

var (
	fileName       = "lorem.txt"
	awsTestDirPath = "unit_tests/"
	keyOnAws       = awsTestDirPath + fileName
	localFilePath  string
)

//func Test_S3_Init(t *testing.T) {
//	SetTesting("../.env")
//	workingDir, _ := os.Getwd()
//	localFilePath = workingDir + string(os.PathSeparator) + fileName
//}

/*
Test_Multi_Part_Uploads tests multipart uploads
	1. Run tests for a successful multipart upload.
	2. Run tests for an aborted multipart upload.
*/
//func Test_Multi_Part_Uploads(t *testing.T) {
//	multipartUploadCompletionTest(t)
//	multipartUploadAbortionTest(t)
//}

/*
multipartUploadCompletionTest tests for successful completion of multipart upload
	1. Verify the file does not exist on S3.
	2. Start and finish a multipart upload to S3, verifying no errors.
	3. Verify the file exists on S3.
	4. Remove the file from S3.
	5. Verify it was removed from S3.
*/
func multipartUploadCompletionTest(t *testing.T) {
	verifyFileIsNotOnS3(keyOnAws, t)

	successfulMultipartUploadForTest(t)

	verifyFileIsOnS3(keyOnAws, t)

	removeFileFromS3(keyOnAws, t)
}

/*
multipartUploadAbortionTest tests for successful abortion of multipart upload
	1. Verify the file does not exist on S3.
	2. Start a multipart upload to S3.
	3. Abort the upload to S3 before completion.
	4. Verify no abortion errors.
	5. Verify the file does not exist on S3.
*/
func multipartUploadAbortionTest(t *testing.T) {
	verifyFileIsNotOnS3(keyOnAws, t)

	failedMultipartUploadForTest(t)

	verifyFileIsNotOnS3(keyOnAws, t)
}

/*startUploadForTest starts a multipart upload for unit tests and returns the aws object key, upload id,
buffer, and file size*/
func startUploadForTest(t *testing.T) (string, string, []byte, int64) {
	file, err := os.Open(localFilePath)
	assert.Nil(t, err)
	defer file.Close()
	fileInfo, _ := file.Stat()
	size := fileInfo.Size()
	buffer := make([]byte, size)
	fileType := http.DetectContentType(buffer)
	file.Read(buffer)

	key := awsTestDirPath + fileName
	input := &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(Env.BucketName),
		Key:         aws.String(key),
		ContentType: aws.String(fileType),
	}

	resp, err := svc.StartMultipartUpload(input)
	assert.Nil(t, err)

	return key, aws.StringValue(resp.UploadId), buffer, size
}

/*successfulMultipartUploadForTest will execute a successful multipart upload for unit tests and verify there were
no errors*/
func successfulMultipartUploadForTest(t *testing.T) {
	key, uploadID, buffer, size := startUploadForTest(t)

	var curr, partLength int64
	var remaining = size
	var completedParts []*s3.CompletedPart
	partNumber := 1
	for curr = 0; remaining != 0; curr += partLength {
		if remaining < MaxMultiPartSizeForTest {
			partLength = remaining
		} else {
			partLength = MaxMultiPartSizeForTest
		}
		completedPart, uploadPartErr := svc.UploadPartOfMultiPartUpload(key, uploadID, buffer[curr:curr+partLength], partNumber)
		if uploadPartErr != nil {
			cancelErr := svc.CancelMultipartUpload(key, uploadID)
			assert.Nil(t, CollectErrors([]error{uploadPartErr, cancelErr}))
		}
		remaining -= partLength
		partNumber++
		completedParts = append(completedParts, completedPart)
	}

	_, err := svc.FinishMultipartUpload(key, uploadID, completedParts)
	assert.Nil(t, err)
}

/*failedMultipartUploadForTest will initiate a multipart upload for unit tests, but will abort it before the upload
is complete.  It will verify there were no errors aborting the upload.*/
func failedMultipartUploadForTest(t *testing.T) {
	key, uploadID, buffer, size := startUploadForTest(t)

	var curr, partLength int64
	partNumber := 1
	curr = 0
	partLength = size

	_, uploadPartErr := svc.UploadPartOfMultiPartUpload(key, uploadID, buffer[curr:curr+partLength], partNumber)
	assert.Nil(t, uploadPartErr)
	cancelErr := svc.CancelMultipartUpload(key, uploadID)
	assert.Nil(t, cancelErr)
}

/*verifyFileIsNotOnS3 checks that the file is not on S3*/
func verifyFileIsNotOnS3(keyOnAws string, t *testing.T) {
	newS3Session()

	_, errGetObject := svc.s3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(Env.BucketName),
		Key:    aws.String(keyOnAws),
	})
	assert.NotNil(t, errGetObject)
	assert.Contains(t, errGetObject.Error(), s3.ErrCodeNoSuchKey)
}

/*verifyFileIsOnS3 checks that the file is on S3*/
func verifyFileIsOnS3(keyOnAws string, t *testing.T) {
	newS3Session()

	_, errGetObject := svc.s3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(Env.BucketName),
		Key:    aws.String(keyOnAws),
	})
	assert.Nil(t, errGetObject)
}

/*removeFileFromS3 removes the file from S3 and then verifies it is no longer on S3.*/
func removeFileFromS3(keyOnAws string, t *testing.T) {
	newS3Session()

	_, err := svc.s3.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(Env.BucketName),
		Key:    aws.String(keyOnAws),
	})
	assert.Nil(t, err)
	verifyFileIsNotOnS3(keyOnAws, t)
}

/*verifyLocalFileExists verifies that the local test file exists*/
func verifyLocalFileExists(localFilePath string, t *testing.T) {
	_, err := os.Stat(localFilePath)
	assert.Nil(t, err)
}

/*verifyLocalFileDoesNotExist verifies that the local test file does not exist*/
func verifyLocalFileDoesNotExist(localFilePath string, t *testing.T) {
	_, err := os.Stat(localFilePath)
	assert.NotNil(t, err)
}

/*deleteLocalTestFile deletes the local test file and verifies there were no errors deleting it.*/
func deleteLocalTestFile(localFilePath string, t *testing.T) {
	err := os.Remove(localFilePath)
	assert.Nil(t, err)
}

/*downloadTestFile downloads the test file and stores it locally.*/
func downloadTestFile(localFilePath string, url string, t *testing.T) (err error) {

	// Create the file
	out, err := os.Create(localFilePath)
	assert.Nil(t, err)

	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	assert.Nil(t, err)

	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		assert.Fail(t, fmt.Sprintf("download failed due to bad status from server: %s", resp.Status))
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	assert.Nil(t, err)

	return nil
}
