package models

import (
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

func DeleteAccountsForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeleteAccountsForTest method on test database")
	} else {
		DB.Exec("DELETE from accounts;")
	}
}

func DeleteExpiredAccountsForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeleteExpiredAccountsForTest method on test database")
	} else {
		DB.Exec("DELETE from expired_accounts;")
	}
}

func DeleteUpgradesForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeleteUpgradesForTest method on test database")
	} else {
		DB.Exec("DELETE from upgrades;")
	}
}

func DeleteRenewalsForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeleteRenewalsForTest method on test database")
	} else {
		DB.Exec("DELETE from renewals;")
	}
}

func DeleteFilesForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeleteFilesForTest method on test database")
	} else {
		DB.Exec("DELETE from files;")
	}
}

func DeleteCompletedFilesForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeleteCompletedFilesForTest method on test database")
	} else {
		DB.Exec("DELETE from completed_files;")
	}
}

func DeleteCompletedUploadIndexesForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeleteCompletedUploadIndexesForTest method on test database")
	} else {
		DB.Exec("DELETE from completed_upload_indices;")
	}
}

func DeleteStripePaymentsForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeleteStripePaymentsForTest method on test database")
	} else {
		DB.Exec("DELETE from stripe_payments;")
	}
}

func DeletePublicSharesForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeletePublicSharesForTest method on test database")
	} else {
		DB.Exec("DELETE from public_shares;")
	}
}

func CreateTestPublicShare(t *testing.T) PublicShare {
	ps := CreatePublicShareObj()
	assert.Nil(t, DB.Create(&ps).Error)
	return ps
}

func CreatePublicShareObj() PublicShare {
	shortID, _ := shortid.Generate()
	return PublicShare{
		PublicID:    shortID,
		ViewsCount:  0,
		Title:       "LoremTitle",
		Description: "Sed felis eget velit aliquet sagittis id consectetur purus ut. Sed libero enim sed faucibus turpis. Leo urna molestie at elementum. Sed egestas egestas fringilla phasellus faucibus.",
		FileID:      utils.GenerateFileHandle(),
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
