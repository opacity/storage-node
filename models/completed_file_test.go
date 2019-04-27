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

func Test_GetAllExpiredCompletedFiles(t *testing.T) {
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

	f, err := GetAllExpiredCompletedFiles(time.Date(2011, 1, 1, 12, 0, 0, 0, time.UTC))
	assert.Nil(t, err)

	expected := []string{"foo1", "foo2"}
	assert.True(t, reflect.DeepEqual(f, expected))
}

func Test_DeleteAllCompletedFiles(t *testing.T) {
	s := CompletedFile{
		FileID:    "foo4",
		ExpiredAt: time.Date(2001, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	assert.Nil(t, DB.Create(&s).Error)
	s = CompletedFile{
		FileID:    "foo5",
		ExpiredAt: time.Date(2002, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	assert.Nil(t, DB.Create(&s).Error)

	f, _ := GetAllExpiredCompletedFiles(time.Date(2003, 1, 1, 12, 0, 0, 0, time.UTC))
	assert.True(t, len(f) == 2)

	assert.Nil(t, DeleteAllCompletedFiles([]string{"foo4", "foo5"}))

	f, _ = GetAllExpiredCompletedFiles(time.Date(2003, 1, 1, 12, 0, 0, 0, time.UTC))
	assert.True(t, len(f) == 0)
}

func Test_DeleteAllCompletedFiles(t *testing.T) {
	s := CompletedFile{
		FileID:         "foo6",
		FileSizeInByte: 150,
	}
	assert.Nil(t, DB.Create(&s).Error)
	s = CompletedFile{
		FileID:         "foo7",
		FileSizeInByte: 210,
	}
	assert.Nil(t, DB.Create(&s).Error)

	v, err := GetTotalFileSizeInByte()

	assert.Nil(t, err)
	assert.True(t, v == int64(150+210))
}
