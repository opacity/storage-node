package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Init_Completed_Upload_Index(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.TestDatabaseURL)
}

func Test_GetCompletedUploadProgress(t *testing.T) {
	assert.Nil(t, CompletedUploadIndex("test_bar1", 1))
	assert.Nil(t, CompletedUploadIndex("test_bar1", 2))
	assert.Nil(t, CompletedUploadIndex("test_bar1", 3))

	c, err := GetCompletedUploadProgress("test_bar1")
	assert.Nil(t, err)
	assert.Equal(t, 3, c)
}

func Test_DeleteCompletedUploadIndexes(t *testing.T) {
	assert.Nil(t, CompletedUploadIndex("test_bar2", 1))
	assert.Nil(t, CompletedUploadIndex("test_bar2", 2))
	assert.Nil(t, CompletedUploadIndex("test_bar3", 2))

	assert.Nil(t, DeleteCompletedUploadIndexes("test_bar2"))

	c, err := GetCompletedUploadProgress("test_bar3")
	assert.Nil(t, err)
	assert.Equal(t, 1, c)
}
