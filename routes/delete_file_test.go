package routes

import (
	"crypto/ecdsa"
	"encoding/json"
	"net/http"
	"testing"

	"bytes"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func testSetupDeleteFiles() {
	utils.SetTesting("../.env")
	models.Connect(utils.Env.DatabaseURL)
}

func Test_Init_Delete_Files(t *testing.T) {
	testSetupDeleteFiles()
	gin.SetMode(gin.TestMode)
}

func Test_Successful_File_Deletion_Request(t *testing.T) {
	models.DeleteAccountsForTest(t)
	models.DeleteCompletedFilesForTest(t)
	models.DeleteFilesForTest(t)
	account, fileID, privateKey := createAccountAndUploadFile(t)

	checkPrerequisites(t, account, fileID)

	deleteFileObject := deleteFileObj{
		FileID: fileID,
	}

	marshalledReq, _ := json.Marshal(deleteFileObject)
	reqBody := bytes.NewBuffer(marshalledReq)

	verificationBody := setupVerificationWithPrivateKeyForTest(t, reqBody.String(), privateKey)

	request := deleteFileReq{
		verification: verificationBody,
		RequestBody:  reqBody.String(),
	}

	w := deleteFileHelperForTest(t, request)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	updatedAccount, err := models.GetAccountById(account.AccountID)
	// check that StorageUsed has been deducted after deletion
	assert.True(t, updatedAccount.StorageUsed == defaultStorageUsedForTest)
	// check that object is not on S3 anymore
	assert.False(t, utils.DoesDefaultBucketObjectExist(models.GetFileMetadataKey(fileID)))
	assert.False(t, utils.DoesDefaultBucketObjectExist(models.GetFileDataKey(fileID)))
	// check that completed file row in SQL table is gone
	completedFile, err := models.GetCompletedFileByFileID(fileID)
	assert.NotNil(t, err)
	assert.NotEqual(t, fileID, completedFile.FileID)
}

func checkPrerequisites(t *testing.T, account models.Account, fileID string) {
	// check that StorageUsed has increased after the upload
	assert.True(t, account.StorageUsed > defaultStorageUsedForTest)
	// check that object exists on S3
	assert.True(t, utils.DoesDefaultBucketObjectExist(models.GetFileMetadataKey(fileID)))
	assert.True(t, utils.DoesDefaultBucketObjectExist(models.GetFileDataKey(fileID)))
	// check that a completed file entry exists
	completedFile, err := models.GetCompletedFileByFileID(fileID)
	assert.Nil(t, err)
	// check that the FileSizeInBytes matches the size on S3
	assert.True(t, utils.GetDefaultBucketObjectSize(models.GetFileDataKey(fileID)) == completedFile.FileSizeInByte)
	filesInDB := []models.File{}
	models.DB.Where("file_id = ?", fileID).Find(&filesInDB)
	// check that there is no "files" row in SQL associated with this ID
	assert.Equal(t, 0, len(filesInDB))
}

func createAccountAndUploadFile(t *testing.T) (models.Account, string, *ecdsa.PrivateKey) {
	privateKey, err := utils.GenerateKey()

	initBody := ReturnValidInitUploadFileBodyForTest(t)
	initReq := ReturnValidInitUploadFileReqForTest(t, initBody, privateKey)

	uploadBody := UploadFileObj{
		FileHandle: initBody.FileHandle,
		PartIndex:  initBody.EndIndex,
	}

	chunkData := ReturnChunkDataForTest(t)

	assert.Nil(t, err)
	request := ReturnValidUploadFileReqForTest(t, uploadBody, privateKey)
	request.ChunkData = string(chunkData)
	accountID, err := utils.HashString(request.PublicKey)
	assert.Nil(t, err)
	account := CreatePaidAccountForTest(accountID, t)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	initializeUpload(initReq, c)

	c, _ = gin.CreateTestContext(httptest.NewRecorder())

	var fileToUpload bytes.Buffer
	fileToUpload.Write(chunkData)

	err = uploadChunk(uploadBody, request, fileToUpload, c)
	assert.Nil(t, err)

	uploadStatusObj := UploadStatusObj{
		FileHandle: initBody.FileHandle,
	}

	uploadStatusReq := ReturnValidUploadStatusReqForTest(t, uploadStatusObj, privateKey)

	w := uploadStatusFileHelperForTest(t, uploadStatusReq)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	updatedAccount, err := models.GetAccountById(account.AccountID)

	assert.Nil(t, err)

	return updatedAccount, uploadBody.FileHandle, privateKey
}

func deleteFileHelperForTest(t *testing.T, request deleteFileReq) *httptest.ResponseRecorder {
	abortIfNotTesting(t)

	router := returnEngine()
	v1 := returnV1Group(router)
	v1.POST(DeletePath, DeleteFileHandler())

	marshalledReq, _ := json.Marshal(request)
	reqBody := bytes.NewBuffer(marshalledReq)

	// Create the mock request you'd like to test. Make sure the second argument
	// here is the same as one of the routes you defined in the router setup
	// block!
	req, err := http.NewRequest(http.MethodPost, v1.BasePath()+DeletePath, reqBody)
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
	}

	// Create a response recorder so you can inspect the response
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	return w
}
