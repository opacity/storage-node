package routes

import (
	"crypto/ecdsa"
	"testing"
	"net/http"

	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_Upload_Status(t *testing.T) {
	setupTests(t)
	cleanUpBeforeTest(t)
}

func Test_CheckWithAccountNoExist(t *testing.T) {
	_, privateKey := generateValidateAccountId(t)
	req, _ := generateRequest(privateKey)

	w := httpPostRequestHelperForTest(t, UploadStatusPath, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "no account with that id")
}

func Test_CheckFileNotFound(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	CreatePaidAccountForTest(t, accountId)

	req, uploadObj := generateRequest(privateKey)
	
	w := httpPostRequestHelperForTest(t, UploadStatusPath, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "no file with that id")
}

func Test_CheckFileIsCompleted(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	CreatePaidAccountForTest(t, accountId)

	req, uploadObj := generateRequest(privateKey)

	compeletedFile := CompletedFile{
		FileID:         uploadObj.FileHandle,
		FileSizeInByte: 100,
		ModifierHash: utils.RandHexString(64),
	}
	assert.Nil(t, models.DB.Create(&compeletedFile).Error)
	assert.Nil(t, utils.SetDefaultBucketObject(models.GetFileDataKey(uploadObj.FileHandle)), "hello world!")

	w := httpPostRequestHelperForTest(t, UploadStatusPath, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "File is uploaded")

	// clean up
	utils.DeleteDefaultBucketObject(models.GetFileDataKey(uploadObj.FileHandle))
}

func Test_MissingIndexes(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	CreatePaidAccountForTest(t, accountId)

	req, uploadObj := generateRequest(privateKey)
	modifiedHash, _ := createModifierHash(privateKey.PublicKey, uploadObj.FileHandle, nil)
	file := File{
		FileID: uploadObj.FileHandle,
		EndIndex: 5,
		ModifierHash: modifiedHash,
	}
	assert.Nil(t, models.DB.Create(&file).Error)
	assert.Nil(t, modles.CreateCompletedUploadIndex(uploadObj.FileHandle, 1, "a"))
	assert.Nil(t, modles.CreateCompletedUploadIndex(uploadObj.FileHandle, 4, "a"))
	assert.Nil(t, modles.CreateCompletedUploadIndex(uploadObj.FileHandle, 2, "a"))

	w := httpPostRequestHelperForTest(t, UploadStatusPath, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "abc")
}

func Test_IncorrectPermission(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	CreatePaidAccountForTest(t, accountId)

	req, uploadObj := generateRequest(privateKey)
	file := File{
		FileID: uploadObj.FileHandle,
		EndIndex: 10,
		ModifierHash: utils.RandHexString(64),
	}
	assert.Nil(t, models.DB.Create(&file).Error)

	w := httpPostRequestHelperForTest(t, UploadStatusPath, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "you are not authorized to modify this file")
}

func generateRequest(privateKey *ecdsa.PrivateKey) (UploadStatusReq, UploadStatusObj) {
	uploadStatusObj := UploadStatusObj{
		FileHandle: utils.GenerateFileHandle(),
	} 
	v, b := returnValidVerificationAndRequestBody(t, uploadStatusObj, privateKey)
	req := UploadStatusReq{
		verification: v,
		requestBody: b,
	}
	return req, uploadStatusObj
}
