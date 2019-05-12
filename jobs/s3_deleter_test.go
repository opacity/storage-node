package jobs

import (
	"testing"
	"time"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_S3_Deleter(t *testing.T) {
	utils.SetTesting("../.env")
	models.Connect(utils.Env.DatabaseURL)
}

func Test_DeleteAllExpiredCompletedFiles(t *testing.T) {
	s1FileID := "test/foo1"
	s2FileID := "test/foo2"
	s3FileID := "test/foo3"

	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling deleteAccounts method on test database")
	} else {
		models.DB.Exec("DELETE from completed_files;")
	}
	s := models.CompletedFile{
		FileID:    s1FileID,
		ExpiredAt: time.Date(2009, 1, 1, 12, 0, 0, 0, time.UTC), // past
	}
	assert.Nil(t, models.DB.Create(&s).Error)
	assert.Nil(t, utils.SetDefaultBucketObject(models.GetFileMetadataKey(s1FileID), "foo1"))

	s = models.CompletedFile{
		FileID:    s2FileID,
		ExpiredAt: time.Date(2010, 1, 1, 12, 0, 0, 0, time.UTC), // past
	}
	assert.Nil(t, models.DB.Create(&s).Error)
	assert.Nil(t, utils.SetDefaultBucketObject(models.GetFileMetadataKey(s2FileID), "foo2"))

	s = models.CompletedFile{
		FileID:    s3FileID,
		ExpiredAt: time.Now().AddDate(0, 1, 0), // future
	}
	assert.Nil(t, models.DB.Create(&s).Error)
	assert.Nil(t, utils.SetDefaultBucketObject(models.GetFileMetadataKey(s3FileID), "foo3"))

	testSubject := s3Deleter{}
	testSubject.Run()

	result := []models.CompletedFile{}
	assert.Nil(t, models.DB.Find(&result).Error)

	assert.Equal(t, 1, len(result))
	assert.Equal(t, s3FileID, result[0].FileID)

	assert.False(t, utils.DoesDefaultBucketObjectExist(models.GetFileMetadataKey(s1FileID)))
	assert.False(t, utils.DoesDefaultBucketObjectExist(models.GetFileMetadataKey(s2FileID)))
	assert.True(t, utils.DoesDefaultBucketObjectExist(models.GetFileMetadataKey(s3FileID)))

	assert.Nil(t, utils.DeleteDefaultBucketObject(models.GetFileMetadataKey(s3FileID)))
}
