package models

import (
	"testing"
	"time"

	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_S3_Life_Cycle(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.DatabaseURL)
}
func Test_Update(t *testing.T) {
	objectName := "abc"
	s := S3ObjectLifeCycle{
		ObjectName:  objectName,
		ExpiredTime: time.Date(2009, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	assert.Nil(t, DB.Create(&s).Error)

	assert.Nil(t, ExpireObject(objectName))

	s = S3ObjectLifeCycle{}
	assert.Nil(t, DB.Where("object_name = ?", objectName).First(&s).Error)
	assert.True(t, time.Now().Sub(s.ExpiredTime).Minutes() < 10.0)
}

func Test_Create(t *testing.T) {
	objectName := "abc1234"
	assert.Nil(t, ExpireObject(objectName))

	s := S3ObjectLifeCycle{}
	assert.Nil(t, DB.Where("object_name = ?", objectName).First(&s).Error)

	assert.True(t, time.Now().Sub(s.ExpiredTime).Minutes() < 10.0)
}
