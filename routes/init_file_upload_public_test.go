package routes

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_File_Upload_Public(t *testing.T) {
	utils.SetTesting("../.env")
	models.Connect(utils.Env.DatabaseURL)
	models.SetTestPlans()
	gin.SetMode(gin.TestMode)
}

func Test_CreateInitFileUploadPulic(t *testing.T) {
	completedFile := models.CompletedFile{
		FileID:         utils.GenerateFileHandle(),
		ModifierHash:   utils.GenerateFileHandle(),
		FileSizeInByte: 150,
		StorageType:    models.S3,
	}
	assert.Nil(t, models.DB.Create(&completedFile).Error)

	accountID, privateKey := generateValidateAccountId(t)
	CreatePaidAccountForTest(t, accountID)

	fileSizeInByte := int64(2400)
	req, uploadObj := createValidInitFileUploadRequest(t, fileSizeInByte, 1, privateKey)

	form := map[string]string{
		"metadata": "abc",
	}
	formFile := map[string]string{
		"metadata": "abc_file",
	}
	w := httpPostFormRequestHelperForTest(t, InitUploadPublicPath, &req, form, formFile, "v2")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Public file is init. Please continue to upload")

	_, err := models.GetFileById(uploadObj.FileHandle)
	assert.Nil(t, err)
}
