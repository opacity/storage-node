package routes

import (
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

	uploadStatusObj := UploadStatusReq{
		FileHandle: utils.GenerateFileHandle(),
	} 
	v, b := returnValidVerificationAndRequestBody(t, uploadStatusObj, privateKey)
	req := UploadStatusReq{
		verification v,
		requestBody b,
	}

	w := httpPostRequestHelperForTest(t, UploadStatusPath, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func Test_CheckFileNotCompleted(t *testing.T) {
}

func Test_CheckFileIsCompleted(t *testing.T) {
}

func Test_MissingIndexes(t *testing.T) {
}
