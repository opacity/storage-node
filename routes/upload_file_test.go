package routes

import (
	"crypto/ecdsa"
	"testing"
	"net/http"
	"fmt"

	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
	"github.com/opacity/storage-node/models"
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

	w := UploadFileHelperForTest(t, request)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func Test_Upload_File_Without_Init(t *testing.T) {
	_, privateKey := generateValidateAccountId(t)
	uploadObj :=  ReturnValidUploadFileBodyForTest(t)
	request := ReturnValidUploadFileReqForTest(t, uploadObj, privateKey)

	w := UploadFileHelperForTest(t, request)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), fmt.Sprintf("no file with that id: %s", uploadObj.FileHandle))
}

func Test_Upload_Part_Of_File(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	account := CreatePaidAccountForTest(t, accountId)
	fileId := initFileUpload(t, 2, privateKey)

	count, _ := models.GetCompletedUploadProgress(fileId)
	assert.Equal(t, 0, count)

	uploadObj := ReturnValidUploadFileBodyForTest(t)
	uploadObj.FileHandle = fileId
	request := ReturnValidUploadFileReqForTest(t, uploadObj, privateKey)

	w := UploadFileHelperForTest(t, request)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Chunk is uploaded")
	
	count, _ = models.GetCompletedUploadProgress(fileId)
	assert.Equal(t, 1, count)
}

func Test_Upload_Completed_Of_File(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	account := CreatePaidAccountForTest(t, accountId)
	fileId := initFileUpload(t, 2, privateKey)

	uploadObj := ReturnValidUploadFileBodyForTest(t)
	uploadObj.FileHandle = fileId
	request1 := ReturnValidUploadFileReqForTest(t, uploadObj, privateKey)

	uploadObj.PartIndex = 2
	request2 := ReturnValidUploadFileReqForTest(t, uploadObj, privateKey)

	requests := []UploadFileReq{request1, request2}
	for _, r := range requests {
		w := UploadFileHelperForTest(t, r)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Chunk is uploaded")
	}

	count, _ := models.GetCompletedUploadProgress(fileId)
	assert.Equal(t, 2, count)

	checkStatusReq := createUploadStatusRequest(t, fileId, privateKey)
	w := httpPostRequestHelperForTest(t, UploadStatusPath, checkStatusReq)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "File is uploaded")

	// clean up
	utils.DeleteDefaultBucketObject(models.GetFileDataKey(fileId))
}

func Test_Upload_Completed_No_In_Order(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	account := CreatePaidAccountForTest(t, accountId)
	fileId := initFileUpload(t, 3, privateKey)

	count, _ := models.GetCompletedUploadProgress(fileId)
	assert.Equal(t, 0, count)

	uploadObj := ReturnValidUploadFileBodyForTest(t)
	uploadObj.FileHandle = fileId
	request1 := ReturnValidUploadFileReqForTest(t, uploadObj, privateKey)

	uploadObj.PartIndex = 2
	request2 := ReturnValidUploadFileReqForTest(t, uploadObj, privateKey)

	uploadObj.PartIndex = 3
	request3 := ReturnValidUploadFileReqForTest(t, uploadObj, privateKey)

	requests := []UploadFileReq{request3, request1, request2}
	for _, r := range requests {
		w := UploadFileHelperForTest(t, r)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Chunk is uploaded")
	}

	count, _ := models.GetCompletedUploadProgress(fileId)
	assert.Equal(t, 3, count)

	checkStatusReq := createUploadStatusRequest(t, fileId, privateKey)
	w := httpPostRequestHelperForTest(t, UploadStatusPath, checkStatusReq)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "File is uploaded")

	// clean up
	utils.DeleteDefaultBucketObject(models.GetFileDataKey(fileId))
}

// func Test_Upload_File_Completed_File_Is_Deleted(t *testing.T) {
// 	// TODO: Update tests to work with multipart-form requests
// 	t.Skip()
// 	models.DeleteAccountsForTest(t)
// 	models.DeleteCompletedFilesForTest(t)
// 	models.DeleteFilesForTest(t)

// 	uploadBody := ReturnValidUploadFileBodyForTest(t)
// 	uploadBody.PartIndex = models.FirstChunkIndex

// 	chunkData := ReturnChunkDataForTest(t)
// 	chunkDataPart1 := chunkData[0:utils.MaxMultiPartSizeForTest]
// 	chunkDataPart2 := chunkData[utils.MaxMultiPartSizeForTest:]

// 	privateKey, err := utils.GenerateKey()
// 	assert.Nil(t, err)
// 	request := ReturnValidUploadFileReqForTest(t, uploadBody, privateKey)
// 	request.ChunkData = string(chunkDataPart1)
// 	accountID, err := utils.HashString(request.PublicKey)
// 	assert.Nil(t, err)
// 	account := CreatePaidAccountForTest(t, accountID)

// 	objectKey := InitUploadFileForTest(t, request.PublicKey, uploadBody.FileHandle, models.FirstChunkIndex + 1)
// 	w := UploadFileHelperForTest(t, request)

// 	if w.Code != http.StatusOK {
// 		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
// 	}
// 	uploadBody.PartIndex++

// 	request = ReturnValidUploadFileReqForTest(t, uploadBody, privateKey)
// 	request.ChunkData = string(chunkDataPart2)

// 	w = UploadFileHelperForTest(t, request)

// 	if w.Code != http.StatusOK {
// 		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
// 	}

// 	file, _ := models.GetFileById(uploadBody.FileHandle)
// 	assert.True(t, len(file.FileID) == 0)

// 	updatedAccount, err := models.GetAccountById(account.AccountID)
// 	assert.Nil(t, err)

// 	assert.True(t, updatedAccount.StorageUsedInByte > account.StorageUsedInByte)

// 	completedFile, _ := models.GetCompletedFileByFileID(uploadBody.FileHandle)
// 	assert.Equal(t, completedFile.FileID, uploadBody.FileHandle)

// 	err = utils.DeleteDefaultBucketObject(objectKey)
// 	assert.Nil(t, err)
// }

func initFileUpload(t *testing.T, endIndex int, privateKey *ecdsa.PrivateKey) string {
	req, uploadObj := createValidInitFileUploadRequest(t, 123, endIndex, privateKey)
	w := httpPostFormRequestHelperForTest(t, InitUploadPath, &req, make(map[string]string), make(map[string]string))

	assert.Equal(t, http.StatusOK, w.Code)

	return uploadObj.FileHandle
}
