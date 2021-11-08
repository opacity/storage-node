package jobs

import (
	"testing"
	"time"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_Expired_Completed_Files_Deleter(t *testing.T) {
	utils.SetTesting("../.env")
	models.Connect(utils.Env.DatabaseURL)
	models.SetTestPlans()
}

func Test_DeleteAllExpiredCompletedFilesS3(t *testing.T) {
	s1FileID := utils.GenerateFileHandle()
	s2FileID := utils.GenerateFileHandle()
	s3FileID := utils.GenerateFileHandle()

	models.DeleteCompletedFilesForTest(t)

	s := models.CompletedFile{
		FileID:         s1FileID,
		ModifierHash:   utils.GenerateFileHandle(),
		ExpiredAt:      time.Date(2009, 1, 1, 12, 0, 0, 0, time.UTC), // past
		StorageType:    utils.S3,
		FileSizeInByte: 123,
	}
	assert.Nil(t, models.DB.Create(&s).Error)
	assert.Nil(t, utils.SetDefaultBucketObject(models.GetFileMetadataKey(s1FileID), "foo1", "", utils.S3))

	s = models.CompletedFile{
		FileID:         s2FileID,
		ModifierHash:   utils.GenerateFileHandle(),
		ExpiredAt:      time.Now().Add(-24 * time.Hour * 61), // past
		StorageType:    utils.S3,
		FileSizeInByte: 123,
	}
	assert.Nil(t, models.DB.Create(&s).Error)
	assert.Nil(t, utils.SetDefaultBucketObject(models.GetFileMetadataKey(s2FileID), "foo2", "", utils.S3))

	s = models.CompletedFile{
		FileID:         s3FileID,
		ModifierHash:   utils.GenerateFileHandle(),
		ExpiredAt:      time.Now().Add(-24 * time.Hour * 59), // past, but not old enough to be deleted
		StorageType:    utils.S3,
		FileSizeInByte: 123,
	}
	assert.Nil(t, models.DB.Create(&s).Error)
	assert.Nil(t, utils.SetDefaultBucketObject(models.GetFileMetadataKey(s3FileID), "foo3", "", utils.S3))

	testSubject := expiredCompletedFilesDeleter{}
	testSubject.Run()

	result := []models.CompletedFile{}
	assert.Nil(t, models.DB.Find(&result).Error)

	assert.Equal(t, 1, len(result))
	assert.Equal(t, s3FileID, result[0].FileID)

	assert.False(t, utils.DoesDefaultBucketObjectExist(models.GetFileMetadataKey(s1FileID), utils.S3))
	assert.False(t, utils.DoesDefaultBucketObjectExist(models.GetFileMetadataKey(s2FileID), utils.S3))
	assert.True(t, utils.DoesDefaultBucketObjectExist(models.GetFileMetadataKey(s3FileID), utils.S3))

	assert.Nil(t, utils.DeleteDefaultBucketObject(models.GetFileMetadataKey(s3FileID), utils.S3))
}
