package routes

import (
	"crypto/ecdsa"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_File_Upload(t *testing.T) {
	utils.SetTesting("../.env")
	models.Connect(utils.Env.DatabaseURL)
	gin.SetMode(gin.TestMode)
}

func Test_initFileUploadWithUnpaidAccount(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)

	CreateUnpaidAccountForTest(t, accountId)
	req := createValidInitFileUploadRequest(t, 123, privateKey)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	err := initFileUploadWithRequest(req, c)
	assert.Contains(t, err.Error(), "Account not paid")
}

func Test_initFileUploadWithPaidAccount(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	CreatePaidAccountForTest(t, accountId)

	req := createValidInitFileUploadRequest(t, 123, privateKey)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	err := initFileUploadWithRequest(req, c)
	assert.Nil(t, err)

	file, err := models.GetFileById(req.initFileUploadObj.FileHandle)
	assert.Nil(t, err)
	assert.Equal(t, req.initFileUploadObj.EndIndex, file.EndIndex)
	assert.NotNil(t, file.AwsUploadID)
	assert.NotNil(t, file.AwsObjectKey)
	assert.NotNil(t, file.ModifierHash)
}

func Test_initFileUploadWithoutEnoughSpace(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	account := CreatePaidAccountForTest(t, accountId)

	fileSizeInByte := (float64(account.StorageLimit) - account.StorageUsed) * float64(1e9) + 1.0
	req := createValidInitFileUploadRequest(t, fileSizeInByte  , privateKey)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	err := initFileUploadWithRequest(req, c)
	assert.Nil(t, err)

	file, err := models.GetFileById(req.initFileUploadObj.FileHandle)
	assert.Nil(t, err)
	assert.Equal(t, "", file.FileID)
}

func createValidInitFileUploadRequest(t *testing.T, fileSizeInByte int64, privateKey *ecdsa.PrivateKey) InitFileUploadReq {
	uploadObj := InitFileUploadObj{
		FileHandle:     utils.RandHexString(64),
		FileSizeInByte: fileSizeInByte,
		EndIndex:       1,
	}
	v, b := returnValidVerificationAndRequestBody(t, uploadObj, privateKey)
	req := InitFileUploadReq{
		verification:      v,
		requestBody:       b,
		initFileUploadObj: uploadObj,
	}
	return req
}
