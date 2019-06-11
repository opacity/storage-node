package routes

import (
	"crypto/ecdsa"
	"net/http/httptest"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_File_Upload(t *testing.T) {
	setupTests(t)
}

func Test_initFileUploadWithUnpaidAccount(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)

	CreateUnpaidAccountForTest(t, accountId)
	req, _ := createValidInitFileUploadRequest(t, 123, privateKey)

	w := httpPostRequestHelperForTest(t, InitUploadPath, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `"paymentStatus":"unpaid"`)
	assert.Contains(t, w.Body.String(), `"invoice"`)
}

func Test_initFileUploadWithPaidAccount(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	CreatePaidAccountForTest(t, accountId)

	req, uploadObj := createValidInitFileUploadRequest(t, 123, privateKey)

	w := httpPostRequestHelperForTest(t, InitUploadPath, req)

	assert.Equal(t, http.StatusOK, w.Code)

	file, err := models.GetFileById(uploadObj.FileHandle)
	assert.Nil(t, err)
	assert.Equal(t, uploadObj.EndIndex, file.EndIndex)
	assert.NotNil(t, file.AwsUploadID)
	assert.NotNil(t, file.AwsObjectKey)
	assert.NotNil(t, file.ModifierHash)

	assert.Nil(t, utils.DeleteDefaultBucketObjectKeys(file.FileID))
}

func Test_initFileUploadWithoutEnoughSpace(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	account := CreatePaidAccountForTest(t, accountId)

	fileSizeInByte := (int64(account.StorageLimit)-(int64(account.StorageUsedInByte)/1e9))*1e9 + 1
	req, uploadObj := createValidInitFileUploadRequest(t, fileSizeInByte, privateKey)
	w := httpPostRequestHelperForTest(t, InitUploadPath, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Account does not have enough space")

	_, err = models.GetFileById(uploadObj.FileHandle)
	assert.True(t, gorm.IsRecordNotFoundError(err))
}

func createValidInitFileUploadRequest(t *testing.T, fileSizeInByte int64, privateKey *ecdsa.PrivateKey) (InitFileUploadReq, InitFileUploadObj) {
	uploadObj := InitFileUploadObj{
		FileHandle:     utils.GenerateFileHandle(),
		FileSizeInByte: fileSizeInByte,
		EndIndex:       1,
	}
	v, b := returnValidVerificationAndRequestBody(t, uploadObj, privateKey)
	req := InitFileUploadReq{
		verification:      v,
		requestBody:       b,
	}
	return req, uploadObj
}
