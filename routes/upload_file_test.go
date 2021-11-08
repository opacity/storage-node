package routes

import (
	"crypto/ecdsa"
	"fmt"
	"net/http"
	"testing"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_Upload_Files(t *testing.T) {
	setupTests(t)
}

func Test_Upload_File_Bad_Request(t *testing.T) {
	// TODO: Update tests to work with multipart-form requests
	t.Skip()
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	body := ReturnValidUploadFileBodyForTest(t)
	body.PartIndex = 0
	request := ReturnValidUploadFileReqForTest(t, body, privateKey)

	w := UploadFileHelperForTest(t, request, UploadPath, "v1")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func Test_Upload_File_Without_Init(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	CreatePaidAccountForTest(t, accountId)

	uploadObj := ReturnValidUploadFileBodyForTest(t)
	request := ReturnValidUploadFileReqForTest(t, uploadObj, privateKey)

	w := UploadFileHelperForTest(t, request, UploadPath, "v1")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), fmt.Sprintf("no file with that id: %s", uploadObj.FileHandle))
}

func Test_UploadFileLessThanMinSize(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	CreatePaidAccountForTest(t, accountId)
	fileId := initFileUpload(t, 2, privateKey)

	count, _ := models.GetCompletedUploadProgress(fileId)
	assert.Equal(t, 0, count)

	uploadObj := ReturnValidUploadFileBodyForTest(t)
	uploadObj.FileHandle = fileId
	request := ReturnValidUploadFileReqForTest(t, uploadObj, privateKey)
	request.ChunkData = "Hello world!!"

	w := UploadFileHelperForTest(t, request, UploadPath, "v1")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "does not meet min fileSize")

	count, _ = models.GetCompletedUploadProgress(fileId)
	assert.Equal(t, 0, count)
}

func Test_Upload_Part_Of_File(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	CreatePaidAccountForTest(t, accountId)
	fileId := initFileUpload(t, 2, privateKey)

	count, _ := models.GetCompletedUploadProgress(fileId)
	assert.Equal(t, 0, count)

	uploadObj := ReturnValidUploadFileBodyForTest(t)
	uploadObj.FileHandle = fileId
	request := ReturnValidUploadFileReqForTest(t, uploadObj, privateKey)
	request.ChunkData = utils.RandHexString(int(utils.MinMultiPartSize))

	w := UploadFileHelperForTest(t, request, UploadPath, "v1")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Chunk is uploaded")

	count, _ = models.GetCompletedUploadProgress(fileId)
	assert.Equal(t, 1, count)
}

func Test_Upload_Completed_Of_File(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	CreatePaidAccountForTest(t, accountId)
	fileId := initFileUpload(t, 2, privateKey)

	chunkData1 := utils.RandHexString(int(utils.MinMultiPartSize))
	chunkData2 := utils.RandHexString(2)
	uploadObj := ReturnValidUploadFileBodyForTest(t)
	uploadObj.FileHandle = fileId
	request1 := ReturnValidUploadFileReqForTest(t, uploadObj, privateKey)
	request1.ChunkData = chunkData1

	uploadObj.PartIndex = 2
	request2 := ReturnValidUploadFileReqForTest(t, uploadObj, privateKey)
	request2.ChunkData = chunkData2

	requests := []UploadFileReq{request1, request2}
	for _, r := range requests {
		w := UploadFileHelperForTest(t, r, UploadPath, "v1")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Chunk is uploaded")
	}

	count, _ := models.GetCompletedUploadProgress(fileId)
	assert.Equal(t, 2, count)

	checkStatusReq, _ := createUploadStatusRequest(t, fileId, privateKey)
	w := httpPostRequestHelperForTest(t, UploadStatusPath, "v1", checkStatusReq)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "File is uploaded")

	// read data back:
	data, _ := utils.GetDefaultBucketObject(models.GetFileDataKey(fileId), utils.S3)

	assert.Equal(t, fmt.Sprintf("%s%s", chunkData1, chunkData2), data)

	// clean up
	utils.DeleteDefaultBucketObject(models.GetFileDataKey(fileId))
}

func Test_Upload_Completed_No_In_Order(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	CreatePaidAccountForTest(t, accountId)
	fileId := initFileUpload(t, 3, privateKey)

	count, _ := models.GetCompletedUploadProgress(fileId)
	assert.Equal(t, 0, count)

	chunkData1 := utils.RandHexString(int(utils.MinMultiPartSize))
	chunkData2 := utils.RandHexString(int(utils.MinMultiPartSize))
	chunkData3 := utils.RandHexString(100)

	uploadObj := ReturnValidUploadFileBodyForTest(t)
	uploadObj.FileHandle = fileId
	request1 := ReturnValidUploadFileReqForTest(t, uploadObj, privateKey)
	request1.ChunkData = chunkData1

	uploadObj.PartIndex = 2
	request2 := ReturnValidUploadFileReqForTest(t, uploadObj, privateKey)
	request2.ChunkData = chunkData2

	uploadObj.PartIndex = 3
	request3 := ReturnValidUploadFileReqForTest(t, uploadObj, privateKey)
	request3.ChunkData = chunkData3

	requests := []UploadFileReq{request3, request1, request2}
	for _, r := range requests {
		w := UploadFileHelperForTest(t, r, UploadPath, "v1")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Chunk is uploaded")
	}

	count, _ = models.GetCompletedUploadProgress(fileId)
	assert.Equal(t, 3, count)

	checkStatusReq, _ := createUploadStatusRequest(t, fileId, privateKey)
	w := httpPostRequestHelperForTest(t, UploadStatusPath, "v1", checkStatusReq)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "File is uploaded")

	// read data back:
	data, _ := utils.GetDefaultBucketObject(models.GetFileDataKey(fileId), utils.S3)

	assert.Equal(t, fmt.Sprintf("%s%s%s", chunkData1, chunkData2, chunkData3), data)
	// clean up
	utils.DeleteDefaultBucketObject(models.GetFileDataKey(fileId))
}

func initFileUpload(t *testing.T, endIndex int, privateKey *ecdsa.PrivateKey) string {
	req, uploadObj := createValidInitFileUploadRequest(t, 123, endIndex, privateKey)
	form := map[string]string{
		"metadata": "abc",
	}
	formFile := map[string]string{
		"metadata": "abc_file",
	}

	w := httpPostFormRequestHelperForTest(t, InitUploadPath, &req, form, formFile, "v1")

	assert.Equal(t, http.StatusOK, w.Code)

	return uploadObj.FileHandle
}
