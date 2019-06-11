package routes

import (
	"crypto/ecdsa"
	"testing"
	"net/http"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
	"github.com/opacity/storage-node/models"
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

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func Test_Upload_File_No_Account_Found(t *testing.T) {
	_, privateKey := generateValidateAccountId(t)
	request := ReturnValidUploadFileReqForTest(t, ReturnValidUploadFileBodyForTest(t), privateKey)

	w := UploadFileHelperForTest(t, request)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "no account with that id")
}

func Test_Upload_File_Account_Not_Paid(t *testing.T) {
	accountId, privateKey := generateValidateAccountId(t)
	CreateUnpaidAccountForTest(t, accountId)

	request := ReturnValidUploadFileReqForTest(t, ReturnValidUploadFileBodyForTest(t), privateKey)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	w := UploadFileHelperForTest(t, request)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "invoice")
	assert.Contains(t, w.Body.String(), "cost")
	assert.Contains(t, w.Body.String(), "ethAddress")
	assert.Contains(t, w.Body.String(), "expirationDate")
}

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

// 	assert.True(t, updatedAccount.StorageUsedInByte > account.StorageUsedInByte)

// 	completedFile, _ := models.GetCompletedFileByFileID(uploadBody.FileHandle)
// 	assert.Equal(t, completedFile.FileID, uploadBody.FileHandle)

// 	err = utils.DeleteDefaultBucketObject(objectKey)
// 	assert.Nil(t, err)
// }

func initFileUpload(t *testing.T, privateKey *ecdsa.PrivateKey) string {
	req := createValidInitFileUploadRequest(t, 123, privateKey)
	httpPostRequestHelperForTest(t, InitUploadPath, req)
	return req.initFileUploadObj.FileHandle
}