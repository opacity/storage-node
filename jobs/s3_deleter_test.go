package jobs

import (
	"testing"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_S3_Deleter(t *testing.T) {
	utils.SetTesting("../.env")
	models.Connect(utils.Env.DatabaseURL)
}

func Test_DeleteAllExpiredCompletedFiles(t *testing.T) {
	s := models.CompletedFile{
		FileID:    "foo1",
		ExpiredAt: time.Date(2009, 1, 1, 12, 0, 0, 0, time.UTC), // past
	}
	assert.Nil(t, models.DB.Create(&s).Error)
	s = models.CompletedFile{
		FileID:    "foo2",
		ExpiredAt: time.Date(2010, 1, 1, 12, 0, 0, 0, time.UTC), // past
	}
	assert.Nil(t, models.DB.Create(&s).Error)
	s = models.CompletedFile{
		FileID:    "foo3",
		ExpiredAt: time.Now.AddDate(0, 1, 0), // future
	}
	assert.Nil(t, models.DB.Create(&s).Error)

	testSubject := s3Deleter{}
	testSubject.Run()

	result := []models.CompletedFile{}
	assert.Nil(t, model.DB.Find(&result).Error)

	assert.Equal(t, 1, len(result))
	assert.Equal(t, "foo3", result[0].FileID)
}
