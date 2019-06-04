package routes

import (
	"testing"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_Upload_Files(t *testing.T) {
	setupTests(t)
}

func Test_Upload_File_Bad_Request(t *testing.T) {
	// TODO: Update tests to work with multipart-form requests
	t.Skip()
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	body := ReturnValidUploadFileBodyForTest(t)
	body.PartIndex = 0
	request := ReturnValidUploadFileReqForTest(t, body, privateKey)

	w := UploadFileHelperForTest(t, request)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}
}

// func Test_Upload_File_No_Account_Found(t *testing.T) {
// 	privateKey, err := utils.GenerateKey()
// 	assert.Nil(t, err)
// 	request := ReturnValidUploadFileReqForTest(t, ReturnValidUploadFileBodyForTest(t), privateKey)

// 	w := UploadFileHelperForTest(t, request)

// 	if w.Code != http.StatusNotFound {
// 		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusNotFound, w.Code)
// 	}
// }

//func Test_Upload_File_Account_Not_Paid(t *testing.T) {
//	privateKey, err := utils.GenerateKey()
//	assert.Nil(t, err)
//	request := ReturnValidUploadFileReqForTest(t, ReturnValidUploadFileBodyForTest(t), privateKey)
//	accountID, err := utils.HashString(request.PublicKey)
//	assert.Nil(t, err)
//	CreateUnpaidAccountForTest(accountID, t)
//
//	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
//		return false, nil
//	}
//
//	w := UploadFileHelperForTest(t, request)
//
//	if w.Code != http.StatusForbidden {
//		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusForbidden, w.Code)
//	}
//
//	assert.Contains(t, w.Body.String(), "invoice")
//	assert.Contains(t, w.Body.String(), "cost")
//	assert.Contains(t, w.Body.String(), "ethAddress")
//	assert.Contains(t, w.Body.String(), "expirationDate")
//}

//func Test_Upload_File_Account_Paid_Upload_Starts(t *testing.T) {
//	if err := models.DB.Unscoped().Delete(&models.Account{}).Error; err != nil {
//		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
//	}
//
//	uploadBody := ReturnValidUploadFileBodyForTest(t)
//	uploadBody.ChunkData = string(ReturnChunkDataForTest(t))
//	fileId := uploadBody.FileHandle
//	privateKey, err := utils.GenerateKey()
//	assert.Nil(t, err)
//	request := ReturnValidUploadFileReqForTest(t, uploadBody, privateKey)
//	accountID, err := utils.HashString(request.PublicKey)
//	assert.Nil(t, err)
//	CreatePaidAccountForTest(t, accountID)
//
//	filesInDB := []models.File{}
//	models.DB.Where("file_id = ?", fileId).Find(&filesInDB)
//	assert.Equal(t, 0, len(filesInDB))
//
//	w := UploadFileHelperForTest(t, request)
//
//	if w.Code != http.StatusOK {
//		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
//	}
//
//	filesInDB = []models.File{}
//	models.DB.Where("file_id = ?", fileId).Find(&filesInDB)
//	assert.Equal(t, 1, len(filesInDB))
//
//	assert.NotNil(t, filesInDB[0].AwsUploadID)
//	assert.NotNil(t, filesInDB[0].AwsObjectKey)
//
//	utils.AbortMultiPartUpload(aws.StringValue(filesInDB[0].AwsObjectKey),
//		aws.StringValue(filesInDB[0].AwsUploadID))
//}

//func Test_Upload_File_Account_Paid_Upload_Continues(t *testing.T) {
//	if err := models.DB.Unscoped().Delete(&models.Account{}).Error; err != nil {
//		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
//	}
//
//	uploadBody := ReturnValidUploadFileBodyForTest(t)
//	uploadBody.ChunkData = string(ReturnChunkDataForTest(t))
//	privateKey, err := utils.GenerateKey()
//	assert.Nil(t, err)
//	request := ReturnValidUploadFileReqForTest(t, uploadBody, privateKey)
//	accountID, err := utils.HashString(request.PublicKey)
//	assert.Nil(t, err)
//	CreatePaidAccountForTest(t, accountID)
//
//	w := UploadFileHelperForTest(t, request)
//
//	if w.Code != http.StatusOK {
//		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
//	}
//
//	nextBody := uploadBody
//	nextBody.PartIndex = uploadBody.PartIndex + 1
//	request = ReturnValidUploadFileReqForTest(t, nextBody, privateKey)
//
//	w = UploadFileHelperForTest(t, request)
//
//	if w.Code != http.StatusOK {
//		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
//	}
//
//	filesInDB := []models.File{}
//	models.DB.Where("file_id = ?", uploadBody.FileHandle).Find(&filesInDB)
//	assert.Equal(t, 1, len(filesInDB))
//
//	completedPartsAsArray := filesInDB[0].GetCompletedPartsAsArray()
//	assert.Equal(t, 2, len(completedPartsAsArray))
//
//	utils.AbortMultiPartUpload(aws.StringValue(filesInDB[0].AwsObjectKey),
//		aws.StringValue(filesInDB[0].AwsUploadID))
//}

// func Test_Upload_File_Completed_File_Is_Deleted(t *testing.T) {
// 	// TODO: Update tests to work with multipart-form requests
// 	t.Skip()
// 	models.DeleteAccountsForTest(t)
// 	models.DeleteCompletedFilesForTest(t)
// 	models.DeleteFilesForTest(t)

// 	uploadBody := ReturnValidUploadFileBodyForTest(t)
// 	uploadBody.PartIndex = models.FirstChunkIndex

// 	chunkData := ReturnChunkDataForTest(t)
// 	chunkDataPart1 := chunkData[0:utils.MaxMultiPartSizeForTest]
// 	chunkDataPart2 := chunkData[utils.MaxMultiPartSizeForTest:]

// 	privateKey, err := utils.GenerateKey()
// 	assert.Nil(t, err)
// 	request := ReturnValidUploadFileReqForTest(t, uploadBody, privateKey)
// 	request.ChunkData = string(chunkDataPart1)
// 	accountID, err := utils.HashString(request.PublicKey)
// 	assert.Nil(t, err)
// 	account := CreatePaidAccountForTest(t, accountID)

// 	objectKey := InitUploadFileForTest(t, request.PublicKey, uploadBody.FileHandle, models.FirstChunkIndex + 1)
// 	w := UploadFileHelperForTest(t, request)

// 	if w.Code != http.StatusOK {
// 		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
// 	}
// 	uploadBody.PartIndex++

// 	request = ReturnValidUploadFileReqForTest(t, uploadBody, privateKey)
// 	request.ChunkData = string(chunkDataPart2)

// 	w = UploadFileHelperForTest(t, request)

// 	if w.Code != http.StatusOK {
// 		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
// 	}

// 	file, _ := models.GetFileById(uploadBody.FileHandle)
// 	assert.True(t, len(file.FileID) == 0)

// 	updatedAccount, err := models.GetAccountById(account.AccountID)
// 	assert.Nil(t, err)

// 	assert.True(t, updatedAccount.StorageUsed > account.StorageUsed)

// 	completedFile, _ := models.GetCompletedFileByFileID(uploadBody.FileHandle)
// 	assert.Equal(t, completedFile.FileID, uploadBody.FileHandle)

// 	err = utils.DeleteDefaultBucketObject(objectKey)
// 	assert.Nil(t, err)
// }
