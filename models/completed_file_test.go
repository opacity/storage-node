package models

import (
	"testing"
	"reflect"

	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_Completed_File(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.TestDatabaseURL)
}

func Test_GetAllExpiredFiles(t *testing.T) {
	s := CompletedFile{
		FileID:      "foo1",
		ExpiredTime: time.Date(2009, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	assert.Nil(t, DB.Create(&s).Error)
	s = CompletedFile {
		FileID: "foo2",
		ExpiredTime: time.Date(2010, 1, 1, 12, 0, 0, 0, time.UTC)
	}
	assert.Nil(t, DB.Create(&s).Error)
	s = CompletedFile {
		FileID: "foo3",
		ExpiredTime: time.Date(2012, 1, 1, 12, 0, 0, 0, time.UTC)
	}
	assert.Nil(t, DB.Create(&s).Error)

	f, err := GetAllExpiredFiles(time.Date(2011, 1, 1, 12, 0, 0, 0, time.UTC))
	assert.Nil(err)

	expected := []string{"foo1", "foo2"}
	assert.True(reflect.DeepEqual(f, expected))
}
