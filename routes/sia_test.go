package routes

import (
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_SiaUpload(t *testing.T) {
	t.SkipNow() // no Sia node implemented for Travis
	uploadFileContent := "In blandit pharetra leo, in volutpat turpis pharetra vitae. Etiam finibus id ante in euismod. Donec mollis gravida neque, eget fermentum magna placerat ut. Suspendisse potenti. Nunc orci turpis, ullamcorper eget turpis ultrices, vestibulum ultricies sapien. Sed vitae dictum enim, non ultrices magna. Quisque pellentesque elit a augue placerat, sed ullamcorper sapien feugiat. Ut id massa venenatis, consectetur enim vitae, pulvinar mauris. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia curae;"
	setupTests(t)
	utils.InitSiaClient()

	fileHandle := utils.GenerateFileHandle()
	log.Printf("fileHandle: %s", fileHandle)
	accountID, privateKey := generateValidateAccountId(t)
	log.Printf("accountID: %s", accountID)
	account := CreatePaidAccountForTest(t, accountID)
	account.StorageLocation = models.Sia

	// init-upload
	siaInitUploadObj := InitFileSiaUploadObj{
		GenericFileActionObj: GenericFileActionObj{
			FileHandle: fileHandle,
		},
		FileSizeInByte: 1024,
	}

	v, b := returnValidVerificationAndRequestBody(t, siaInitUploadObj, privateKey)
	req := InitFileSiaUploadReq{
		verification: v,
		requestBody:  b,
	}

	initForm := map[string]string{
		"metadata": "abc",
	}
	initFormFile := map[string]string{
		"metadata": "abc_file",
	}
	w := httpPostFormRequestHelperForTest(t, SiaPathPrefix+InitUploadPath, &req, initForm, initFormFile, "v2")

	log.Print(w.Body.String())
	assert.Equal(t, http.StatusOK, w.Code)

	// upload file
	uploadFormFile := map[string]string{
		"fileData": uploadFileContent,
	}

	genericFileActionObj := GenericFileActionObj{
		FileHandle: fileHandle,
	}
	v, b = returnValidVerificationAndRequestBody(t, genericFileActionObj, privateKey)
	uploadReq := UploadFileSiaReq{
		verification: v,
		requestBody:  b,
	}

	uploadW := httpPostFormRequestHelperForTest(t, SiaPathPrefix+UploadPath, &uploadReq, nil, uploadFormFile, "v2")
	log.Print(uploadW.Body.String())
	assert.Equal(t, http.StatusOK, uploadW.Code)

	// Status
	genericUploadObj := GenericFileActionObj{
		FileHandle: fileHandle,
	}
	v, b = returnValidVerificationAndRequestBody(t, genericUploadObj, privateKey)
	uploadStatusReq := UploadStatusReq{
		verification: v,
		requestBody:  b,
	}

	uploadStatusW := httpPostRequestHelperForTest(t, SiaPathPrefix+UploadStatusPath, "v2", uploadStatusReq)
	log.Print(uploadStatusW.Body.String())
	assert.Equal(t, http.StatusOK, uploadStatusW.Code)

	for strings.Contains(uploadStatusW.Body.String(), "sia file still uploading") {
		uploadStatusW = httpPostRequestHelperForTest(t, SiaPathPrefix+UploadStatusPath, "v2", uploadStatusReq)
		log.Print(uploadStatusW.Body.String())
		time.Sleep(1 * time.Second)
	}

}
