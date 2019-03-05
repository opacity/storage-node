package jobs

import (
	"testing"
	"time"

	"github.com/opacity/storage-node/models"
	"github.com/stretchr/testify/assert"
)

func Test_Delete_S3Object(t *testing.T) {
	models.DB.Create(&models.S3ObjectLifeCycle{
		ObjectName:  "foo",
		ExpiredTime: time.Date(2009, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	models.DB.Create(&models.S3ObjectLifeCycle{
		ObjectName:  "bar",
		ExpiredTime: time.Now().Add(-time.Hour * 1),
	})
	models.DB.Create(&models.S3ObjectLifeCycle{
		ObjectName:  "foobar",
		ExpiredTime: time.Now().Add(time.Hour * 1),
	})

	testSubject := s3ExpireAccess{}
	testSubject.Run()

	result := []models.S3ObjectLifeCycle{}
	assert.Nil(t, models.DB.Find(&result).Error)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, "foobar", result[0].ObjectName)
}
