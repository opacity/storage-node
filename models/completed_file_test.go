package models

import (
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
	DeleteCompletedFilesForTest(t)
	s := CompletedFile{
		FileID:       utils.GenerateFileHandle(),
		ModifierHash: utils.GenerateFileHandle(),
		ExpiredAt:    time.Date(2009, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	firstExpired := s.FileID
	assert.Nil(t, DB.Create(&s).Error)
	s = CompletedFile{
		FileID:       utils.GenerateFileHandle(),
		ModifierHash: utils.GenerateFileHandle(),
		ExpiredAt:    time.Date(2010, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	secondExpired := s.FileID
	assert.Nil(t, DB.Create(&s).Error)
	s = CompletedFile{
		FileID:       utils.GenerateFileHandle(),
		ModifierHash: utils.GenerateFileHandle(),
		ExpiredAt:    time.Date(2012, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	assert.Nil(t, DB.Create(&s).Error)

	f, err := GetAllExpiredCompletedFiles(time.Date(2011, 1, 1, 12, 0, 0, 0, time.UTC))
	assert.Nil(t, err)

	assert.True(t, f[0] == firstExpired || f[0] == secondExpired)
	assert.True(t, f[1] == firstExpired || f[1] == secondExpired)
	assert.True(t, f[0] != f[1])
}

func Test_DeleteAllCompletedFiles(t *testing.T) {
	DeleteCompletedFilesForTest(t)
	s1 := CompletedFile{
		FileID:       utils.GenerateFileHandle(),
		ModifierHash: utils.GenerateFileHandle(),
		ExpiredAt:    time.Date(2001, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	assert.Nil(t, DB.Create(&s1).Error)
	s2 := CompletedFile{
		FileID:       utils.GenerateFileHandle(),
		ModifierHash: utils.GenerateFileHandle(),
		ExpiredAt:    time.Date(2002, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	assert.Nil(t, DB.Create(&s2).Error)

	f, _ := GetAllExpiredCompletedFiles(time.Date(2003, 1, 1, 12, 0, 0, 0, time.UTC))
	assert.True(t, len(f) == 2)

	assert.Nil(t, DeleteAllCompletedFiles([]string{s1.FileID, s2.FileID}))

	f, _ = GetAllExpiredCompletedFiles(time.Date(2003, 1, 1, 12, 0, 0, 0, time.UTC))
	assert.True(t, len(f) == 0)
}

func Test_GetTotalFileSizeInByte(t *testing.T) {
	DeleteCompletedFilesForTest(t)
	s := CompletedFile{
		FileID:         utils.GenerateFileHandle(),
		ModifierHash:   utils.GenerateFileHandle(),
		FileSizeInByte: 150,
	}
	assert.Nil(t, DB.Create(&s).Error)
	s = CompletedFile{
		FileID:         utils.GenerateFileHandle(),
		ModifierHash:   utils.GenerateFileHandle(),
		FileSizeInByte: 210,
	}
	assert.Nil(t, DB.Create(&s).Error)

	v, err := GetTotalFileSizeInByte()

	assert.Nil(t, err)
	assert.Equal(t, int64(360), v)
}

func Test_GetCompletedFileByFileID(t *testing.T) {
	DeleteCompletedFilesForTest(t)
	s := CompletedFile{
		FileID:         utils.GenerateFileHandle(),
		ModifierHash:   utils.GenerateFileHandle(),
		FileSizeInByte: 150,
	}
	assert.Nil(t, DB.Create(&s).Error)
	s = CompletedFile{
		FileID:         utils.GenerateFileHandle(),
		ModifierHash:   utils.GenerateFileHandle(),
		FileSizeInByte: 210,
	}
	assert.Nil(t, DB.Create(&s).Error)

	completedFile, err := GetCompletedFileByFileID(s.FileID)
	assert.True(t, completedFile.FileID == s.FileID)
	assert.True(t, completedFile.FileSizeInByte == s.FileSizeInByte)
	assert.Nil(t, err)
}
