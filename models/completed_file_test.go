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
	SetTestPlans()
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

func Test_UpdateExpiredAt(t *testing.T) {
	DeleteCompletedFilesForTest(t)

	startingExpiredAtTime := time.Now().Add(3 * 24 * time.Hour)

	publicKey := utils.GenerateFileHandle()

	// The first two files should have their expiration dates updated because
	// we will pass in their file handles
	c1 := CompletedFile{
		FileID:         utils.GenerateFileHandle(),
		FileSizeInByte: 150,
		ExpiredAt:      startingExpiredAtTime,
	}
	modifierHash, _ := utils.HashString(publicKey + c1.FileID)
	c1.ModifierHash = modifierHash
	assert.Nil(t, DB.Create(&c1).Error)

	c2 := CompletedFile{
		FileID:         utils.GenerateFileHandle(),
		FileSizeInByte: 210,
		ExpiredAt:      startingExpiredAtTime,
	}
	modifierHash, _ = utils.HashString(publicKey + c2.FileID)
	c2.ModifierHash = modifierHash
	assert.Nil(t, DB.Create(&c2).Error)

	// the next file should not have its expiration date updated because although
	// we are passing in its file handle, the modifier hash won't match
	c3 := CompletedFile{
		FileID:         utils.GenerateFileHandle(),
		FileSizeInByte: 230,
		ExpiredAt:      startingExpiredAtTime,
	}
	// not using its file handle to create the modifier hash
	modifierHash, _ = utils.HashString(publicKey + utils.GenerateFileHandle())
	c3.ModifierHash = modifierHash
	assert.Nil(t, DB.Create(&c3).Error)

	// the next file should not have its expiration date updated because we will
	// not pass in its file handle
	c4 := CompletedFile{
		FileID:         utils.GenerateFileHandle(),
		FileSizeInByte: 250,
		ExpiredAt:      startingExpiredAtTime,
	}
	modifierHash, _ = utils.HashString(publicKey + c4.FileID)
	c4.ModifierHash = modifierHash
	assert.Nil(t, DB.Create(&c4).Error)

	newExpiredAtTime := time.Now().Add(7 * 24 * time.Hour)

	// pass in handles of first two completed files with new ExpiredAt time
	err := UpdateExpiredAt([]string{c1.FileID, c2.FileID, c3.FileID}, publicKey, newExpiredAtTime)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), `affect 3 rows`)
	assert.Contains(t, err.Error(), `affected 2 rows`)

	// check that first two files have their ExpiredAt times changed
	completedFile, err := GetCompletedFileByFileID(c1.FileID)
	assert.Equal(t, newExpiredAtTime.Day(), completedFile.ExpiredAt.Day())
	assert.Nil(t, err)

	completedFile, err = GetCompletedFileByFileID(c2.FileID)
	assert.Equal(t, newExpiredAtTime.Day(), completedFile.ExpiredAt.Day())
	assert.Nil(t, err)

	// check that third file still has the same expired at time as before
	completedFile, err = GetCompletedFileByFileID(c3.FileID)
	assert.Equal(t, startingExpiredAtTime.Day(), completedFile.ExpiredAt.Day())
	assert.Nil(t, err)

	// check that last file still has the same expired at time as before
	completedFile, err = GetCompletedFileByFileID(c4.FileID)
	assert.Equal(t, startingExpiredAtTime.Day(), completedFile.ExpiredAt.Day())
	assert.Nil(t, err)
}
