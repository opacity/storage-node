package models

import (
	"reflect"
	"testing"
	"time"

	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_Completed_File(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.TestDatabaseURL)
}

func Test_GetAllExpiredFiles(t *testing.T) {
	s := CompletedFile{
		FileID:    "foo1",
		ExpiredAt: time.Date(2009, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	assert.Nil(t, DB.Create(&s).Error)
	s = CompletedFile{
		FileID:    "foo2",
		ExpiredAt: time.Date(2010, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	assert.Nil(t, DB.Create(&s).Error)
	s = CompletedFile{
		FileID:    "foo3",
		ExpiredAt: time.Date(2012, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	assert.Nil(t, DB.Create(&s).Error)

	f, err := GetAllExpiredFiles(time.Date(2011, 1, 1, 12, 0, 0, 0, time.UTC))
	assert.Nil(t, err)

	expected := []string{"foo1", "foo2"}
	assert.True(t, reflect.DeepEqual(f, expected))
}
