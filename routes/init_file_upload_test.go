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
	req := createValidInitFileUploadRequest(t, privateKey)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	err := initFileUploadWithRequest(req, c)
	assert.Contains(t, err.Error(), "Account not paid")
}

func Test_initFileUploadWithPaidAccount(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	account := CreatePaidAccountForTest(t, accountId)

	req := createValidInitFileUploadRequest(t, privateKey)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	err := initFileUploadWithRequest(req, c)
	assert.Nil(t, err)

	fileID := req.initFileUploadObj.FileHandle

	file, err := models.GetFileById(fileID)
	assert.Nil(t, err)

	assert.Equal(t, req.initFileUploadObj.EndIndex, file.EndIndex)
	assert.NotNil(t, file.AwsUploadID)
	assert.NotNil(t, file.AwsObjectKey)
	assert.NotNil(t, file.ModifierHash)
	fmt.Printf("file expired date: %v", file.ExpiredAt)
	fmt.Printf("account expired date: %v", account.ExpirationDate())
	assert.Equal(t, account.ExpirationDate(), file.ExpiredAt)
}

func createValidInitFileUploadRequest(t *testing.T, privateKey *ecdsa.PrivateKey) InitFileUploadReq {
	uploadObj := InitFileUploadObj{
		FileHandle:     utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
		FileSizeInByte: 123,
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
