package models

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_Completed_Upload_Index(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.TestDatabaseURL)
}

func Test_GetCompletedUploadProgress(t *testing.T) {
	DeleteCompletedUploadIndexesForTest(t)
	assert.Nil(t, CreateCompletedUploadIndex("test_bar1", 1, "a"))
	assert.Nil(t, CreateCompletedUploadIndex("test_bar1", 2, "b"))
	assert.Nil(t, CreateCompletedUploadIndex("test_bar1", 3, "c"))

	c, err := GetCompletedUploadProgress("test_bar1")
	assert.Nil(t, err)
	assert.Equal(t, 3, c)
}

func Test_DeleteCompletedUploadIndexes(t *testing.T) {
	DeleteCompletedUploadIndexesForTest(t)
	assert.Nil(t, CreateCompletedUploadIndex("test_bar2", 1, "a"))
	assert.Nil(t, CreateCompletedUploadIndex("test_bar2", 2, "b"))
	assert.Nil(t, CreateCompletedUploadIndex("test_bar3", 2, "c"))

	assert.Nil(t, DeleteCompletedUploadIndexes("test_bar2"))

	c, err := GetCompletedUploadProgress("test_bar2")
	assert.Nil(t, err)
	assert.Equal(t, 0, c)

	c, err = GetCompletedUploadProgress("test_bar3")
	assert.Nil(t, err)
	assert.Equal(t, 1, c)
}

func Test_GetCompletedPartsAsArray(t *testing.T) {
	DeleteCompletedUploadIndexesForTest(t)
	assert.Nil(t, CreateCompletedUploadIndex("test_bar4", 2, "b"))
	assert.Nil(t, CreateCompletedUploadIndex("test_bar4", 1, "a"))
	assert.Nil(t, CreateCompletedUploadIndex("test_bar4", 3, "c"))

	l, err := GetCompletedPartsAsArray("test_bar4")
	assert.Nil(t, err)

	assert.Equal(t, 3, len(l))
	assert.Equal(t, 1, int(aws.Int64Value(l[0].PartNumber)))
	assert.Equal(t, "a", aws.StringValue(l[0].ETag))
	assert.Equal(t, 2, int(aws.Int64Value(l[1].PartNumber)))
	assert.Equal(t, "b", aws.StringValue(l[1].ETag))
	assert.Equal(t, 3, int(aws.Int64Value(l[2].PartNumber)))
	assert.Equal(t, "c", aws.StringValue(l[2].ETag))
}

func Test_GetIncompletedIndexAsArray(t *testing.T) {
	DeleteCompletedUploadIndexesForTest(t)
	assert.Nil(t, CreateCompletedUploadIndex("test_bar5", 5, "b"))
	assert.Nil(t, CreateCompletedUploadIndex("test_bar5", 1, "a"))
	assert.Nil(t, CreateCompletedUploadIndex("test_bar5", 3, "c"))

	l, err := GetIncompletedIndexAsArray("test_bar5", 6)

	assert.Nil(t, err)
	assert.Equal(t, 3, len(l))
	assert.Equal(t, int64(2), l[0])
	assert.Equal(t, int64(4), l[1])
	assert.Equal(t, int64(6), l[2])
}
