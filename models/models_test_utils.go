package models

import (
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
	"github.com/teris-io/shortid"
)

func setupModelsTests() {
	utils.SetTesting("../.env")
	Connect(utils.Env.TestDatabaseURL)
}

func DeleteAccountsForTest() {
	if !(utils.IsTestEnv() || utils.IsDebugEnv()) {
		log.Fatalf("should only be calling DeleteAccountsForTest method on test or dev database")
	} else {
		DB.Exec("DELETE from accounts;")
	}
}

func DeleteExpiredAccountsForTest() {
	if !(utils.IsTestEnv() || utils.IsDebugEnv()) {
		log.Fatalf("should only be calling DeleteExpiredAccountsForTest method on test or dev database")
	} else {
		DB.Exec("DELETE from expired_accounts;")
	}
}

func DeleteUpgradesForTest() {
	if !(utils.IsTestEnv() || utils.IsDebugEnv()) {
		log.Fatalf("should only be calling DeleteUpgradesForTest method on test or dev database")
	} else {
		DB.Exec("DELETE from upgrades;")
	}
}

func DeleteRenewalsForTest() {
	if !(utils.IsTestEnv() || utils.IsDebugEnv()) {
		log.Fatalf("should only be calling DeleteRenewalsForTest method on test or dev database")
	} else {
		DB.Exec("DELETE from renewals;")
	}
}

func DeleteFilesForTest() {
	if !(utils.IsTestEnv() || utils.IsDebugEnv()) {
		log.Fatalf("should only be calling DeleteFilesForTest method on test or dev database")
	} else {
		DB.Exec("DELETE from files;")
	}
}

func DeleteCompletedFilesForTest() {
	if !(utils.IsTestEnv() || utils.IsDebugEnv()) {
		log.Fatalf("should only be calling DeleteCompletedFilesForTest method on test or dev database")
	} else {
		DB.Exec("DELETE from completed_files;")
	}
}

func DeleteCompletedUploadIndexesForTest() {
	if !(utils.IsTestEnv() || utils.IsDebugEnv()) {
		log.Fatalf("should only be calling DeleteCompletedUploadIndexesForTest method on test or dev database")
	} else {
		DB.Exec("DELETE from completed_upload_indices;")
	}
}

func DeleteStripePaymentsForTest() {
	if !(utils.IsTestEnv() || utils.IsDebugEnv()) {
		log.Fatalf("should only be calling DeleteStripePaymentsForTest method on test or dev database")
	} else {
		DB.Exec("DELETE from stripe_payments;")
	}
}

func DeletePublicSharesForTest() {
	if !(utils.IsTestEnv() || utils.IsDebugEnv()) {
		log.Fatalf("should only be calling DeletePublicSharesForTest method on test or dev database")
	} else {
		DB.Exec("DELETE from public_shares;")
	}
}

func CreateTestPublicShare() (ps PublicShare, err error) {
	ps = CreatePublicShareObj()
	err = DB.Create(&ps).Error
	return
}

func CreatePublicShareObj() PublicShare {
	shortID, _ := shortid.Generate()
	return PublicShare{
		PublicID:   shortID,
		ViewsCount: 0,
		FileID:     utils.GenerateFileHandle(),
	}
}

func ReturnValidFile() File {
	return File{
		FileID:           utils.GenerateFileHandle(),
		AwsUploadID:      aws.String(utils.GenerateFileHandle()),
		AwsObjectKey:     aws.String(utils.GenerateFileHandle()),
		EndIndex:         10,
		CompletedIndexes: nil,
		ExpiredAt:        time.Date(2009, 1, 1, 12, 0, 0, 0, time.UTC),
		ModifierHash:     utils.GenerateFileHandle(),
	}
}

func returnCompletedPart(partNumber int) *s3.CompletedPart {
	return &s3.CompletedPart{
		ETag:       aws.String(utils.RandSeqFromRunes(32, []rune("abcdef01234567890"))),
		PartNumber: aws.Int64(int64(partNumber)),
	}
}

func MultipartUploadOfSingleChunk(t *testing.T, f *File) (*s3.CompletedPart, error) {
	workingDir, _ := os.Getwd()
	testDir := filepath.Dir(workingDir)
	testDir = testDir + "/test_files"
	localFilePath := testDir + string(os.PathSeparator) + "lorem.txt"

	file, err := os.Open(localFilePath)
	assert.Nil(t, err)
	defer file.Close()
	fileInfo, _ := file.Stat()
	size := fileInfo.Size()
	buffer := make([]byte, size)
	file.Read(buffer)

	awsObjKey := aws.StringValue(f.AwsObjectKey) + "/file"
	key, uploadID, err := utils.CreateMultiPartUpload(awsObjKey)
	if err != nil {
		return nil, err
	}
	err = f.UpdateKeyAndUploadID(key, uploadID)
	if err != nil {
		return nil, err
	}

	completedPart, err := utils.UploadMultiPartPart(awsObjKey, aws.StringValue(f.AwsUploadID),
		buffer, FirstChunkIndex)
	return completedPart, err
}
