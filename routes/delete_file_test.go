package routes

import (
	"crypto/ecdsa"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_Delete_Files(t *testing.T) {
	setupTests(t)
}

func Test_Successful_File_Deletion_Request(t *testing.T) {
	cleanUpBeforeTest(t)

	account, fileIDs, privateKey := createAccountAndUploadFile(t, 1)
	fileID := fileIDs[0]
	checkPrerequisites(t, account, fileID)

	deleteFileObject := deleteFileObj{
		FileID: fileID,
	}

	v, b := returnValidVerificationAndRequestBody(t, deleteFileObject, privateKey)
	request := deleteFileReq{
		verification: v,
		requestBody:  b,
	}

	w := httpPostRequestHelperForTest(t, DeletePath, "v1", request)
	assert.Equal(t, http.StatusOK, w.Code)

	updatedAccount, err := models.GetAccountById(account.AccountID)
	assert.Nil(t, err)
	// check that StorageUsedInByte has been deducted after deletion
	assert.True(t, updatedAccount.StorageUsedInByte == defaultStorageUsedInByteForTest)
	// check that object is not on S3 anymore
	assert.False(t, utils.DoesDefaultBucketObjectExist(models.GetFileMetadataKey(fileID)))
	assert.False(t, utils.DoesDefaultBucketObjectExist(models.GetFileDataKey(fileID)))
	// check that completed file row in SQL table is gone
	completedFile, err := models.GetCompletedFileByFileID(fileID)
	assert.NotNil(t, err)
	assert.NotEqual(t, fileID, completedFile.FileID)
}

func Test_Successful_Multiple_File_Deletion_Request(t *testing.T) {
	cleanUpBeforeTest(t)

	account, fileIDs, privateKey := createAccountAndUploadFile(t, 3)

	for _, fileID := range fileIDs {
		checkPrerequisites(t, account, fileID)
	}

	deleteFilesObject := deleteFilesObj{
		FileIDs: fileIDs,
	}

	v, b := returnValidVerificationAndRequestBody(t, deleteFilesObject, privateKey)
	request := deleteFilesReq{
		verification: v,
		requestBody:  b,
	}

	w := httpPostRequestHelperForTest(t, DeleteV2Path, "v2", request)
	assert.Equal(t, http.StatusOK, w.Code)

	deleteFilesRes := deleteFilesRes{}
	bodyBytes, err := ioutil.ReadAll(w.Body)
	assert.Nil(t, err)

	err = json.Unmarshal(bodyBytes, &deleteFilesRes)
	assert.Nil(t, err)

	assert.Equal(t, map[string]string{}, deleteFilesRes.UnsuccessfulDeletions)

	updatedAccount, err := models.GetAccountById(account.AccountID)
	assert.Nil(t, err)
	// check that StorageUsedInByte has been deducted after deletion
	assert.True(t, updatedAccount.StorageUsedInByte == defaultStorageUsedInByteForTest)
	for _, fileID := range fileIDs {
		// check that object is not on S3 anymore
		assert.False(t, utils.DoesDefaultBucketObjectExist(models.GetFileMetadataKey(fileID)))
		assert.False(t, utils.DoesDefaultBucketObjectExist(models.GetFileDataKey(fileID)))
		// check that completed file row in SQL table is gone
		completedFile, err := models.GetCompletedFileByFileID(fileID)
		assert.NotNil(t, err)
		assert.NotEqual(t, fileID, completedFile.FileID)
	}
}

func checkPrerequisites(t *testing.T, account models.Account, fileID string) {
	// check that StorageUsedInByte has increased after the upload
	assert.True(t, account.StorageUsedInByte > defaultStorageUsedInByteForTest)
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

func createAccountAndUploadFile(t *testing.T, fileCount int) (models.Account, []string, *ecdsa.PrivateKey) {
	accountId, privateKey := generateValidateAccountId(t)
	account := CreatePaidAccountForTest(t, accountId)

	fileIDs := make([]string, 0)
	for i := 0; i < fileCount; i++ {
		initBody := InitFileUploadObj{
			FileHandle:     utils.GenerateFileHandle(),
			EndIndex:       models.FirstChunkIndex,
			FileSizeInByte: 26214400,
		}

		v, b := returnValidVerificationAndRequestBody(t, initBody, privateKey)
		initReq := InitFileUploadReq{
			initFileUploadObj: initBody,
			verification:      v,
			requestBody:       b,
			Metadata:          utils.GenerateFileHandle(),
			MetadataAsFile:    utils.GenerateFileHandle(),
		}

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		initFileUploadWithRequest(initReq, c)

		uploadBody := UploadFileObj{
			FileHandle: initBody.FileHandle,
			PartIndex:  initBody.EndIndex,
		}
		chunkData := ReturnChunkDataForTest(t)
		request := ReturnValidUploadFileReqForTest(t, uploadBody, privateKey)
		request.ChunkData = string(chunkData)
		request.uploadFileObj = uploadBody

		w := UploadFileHelperForTest(t, request, UploadPath, "v1")
		assert.Equal(t, http.StatusOK, w.Code)

		uploadStatusObj := UploadStatusObj{
			FileHandle: initBody.FileHandle,
		}

		v, b = returnValidVerificationAndRequestBody(t, uploadStatusObj, privateKey)
		uploadStatusReq := UploadStatusReq{
			verification: v,
			requestBody:  b,
		}

		w = httpPostRequestHelperForTest(t, UploadStatusPath, "v1", uploadStatusReq)
		assert.Equal(t, http.StatusOK, w.Code)

		fileIDs = append(fileIDs, uploadBody.FileHandle)
	}

	updatedAccount, err := models.GetAccountById(account.AccountID)

	assert.Nil(t, err)

	return updatedAccount, fileIDs, privateKey
}
