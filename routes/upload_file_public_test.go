package routes

import (
	"crypto/ecdsa"
	"net/http"
	"testing"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_UploadFilePublicStorageDoesNotCount(t *testing.T) {
	t.Skip()
	// completedFile := models.CompletedFile{
	// 	FileID:         utils.GenerateFileHandle(),
	// 	ModifierHash:   utils.GenerateFileHandle(),
	// 	FileSizeInByte: 1003,
	// }
	// assert.Nil(t, models.DB.Create(&completedFile).Error)
	accountID, privateKey := generateValidateAccountId(t)
	account := CreatePaidAccountForTest(t, accountID)
	// fileID := completedFile.FileID

	fileUploadObj := initFileUploadPublic(t, 1, privateKey)
	// Just random data, no file on S3 to get
	chunkData := utils.RandHexString(int(utils.MinMultiPartSize))

	uploadObj := ReturnValidUploadFileBodyForTest(t)
	uploadObj.FileHandle = fileUploadObj.FileHandle
	req := ReturnValidUploadFileReqForTest(t, uploadObj, privateKey)
	req.ChunkData = chunkData

	w := UploadFileHelperForTest(t, req, UploadPublicPath, "v2")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Chunk is uploaded")

	checkStatusReq, _ := createUploadStatusRequest(t, fileUploadObj.FileHandle, privateKey)
	checkStatusReq.uploadStatusObj.FileHandle = fileUploadObj.FileHandle
	w = httpPostRequestHelperForTest(t, UploadStatusPublicPath, "v2", checkStatusReq)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "File is uploaded")

	updatedAccount, _ := models.GetAccountById(account.AccountID)
	assert.Equal(t, updatedAccount.StorageUsedInByte, account.StorageUsedInByte)
}

func initFileUploadPublic(t *testing.T, endIndex int, privateKey *ecdsa.PrivateKey) InitFileUploadObj {
	req, uploadObj := createValidInitFileUploadRequest(t, 123, endIndex, privateKey)
	form := map[string]string{
		"metadata": "abc",
	}
	formFile := map[string]string{
		"metadata": "abc_file",
	}

	w := httpPostFormRequestHelperForTest(t, InitUploadPublicPath, &req, form, formFile, "v2")

	assert.Equal(t, http.StatusOK, w.Code)

	return uploadObj
}
