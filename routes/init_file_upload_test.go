package routes

import (
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

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	err := initFileUploadWithRequest(req, c)
	assert.Contains(t, err.Error(), "Account not paid")
}
