package routes

import (
	"testing"

	"net/http"
	"strings"

	"math/big"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func testSetupUploadFiles() {
	utils.SetTesting("../.env")
	models.Connect(utils.Env.DatabaseURL)
}

func Test_Init_Upload_Files(t *testing.T) {
	testSetupUploadFiles()
	gin.SetMode(gin.TestMode)
}

func Test_Upload_File_Bad_Request(t *testing.T) {
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	request := ReturnValidUploadFileReqForTest(t, ReturnValidUploadFileBodyForTest(t), privateKey)
	request.UploadFile.PartIndex = 0

	w := UploadFileHelperForTest(t, request)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}
}

func Test_Upload_File_No_Account_Found(t *testing.T) {
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	request := ReturnValidUploadFileReqForTest(t, ReturnValidUploadFileBodyForTest(t), privateKey)

	w := UploadFileHelperForTest(t, request)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusNotFound, w.Code)
	}
}

func Test_Upload_File_Account_Not_Paid(t *testing.T) {
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	request := ReturnValidUploadFileReqForTest(t, ReturnValidUploadFileBodyForTest(t), privateKey)
	CreateUnpaidAccountForTest(strings.TrimPrefix(request.Address, "0x"), t)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	w := UploadFileHelperForTest(t, request)

	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusForbidden, w.Code)
	}

	assert.Contains(t, w.Body.String(), "invoice")
	assert.Contains(t, w.Body.String(), "cost")
	assert.Contains(t, w.Body.String(), "ethAddress")
	assert.Contains(t, w.Body.String(), "expirationDate")
}

func Test_Upload_File_Account_Paid_Upload_Starts(t *testing.T) {
	if err := models.DB.Unscoped().Delete(&models.Account{}).Error; err != nil {
		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
	}

	uploadBody := ReturnValidUploadFileBodyForTest(t)
	uploadBody.ChunkData = string(ReturnChunkDataForTest(t))
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	request := ReturnValidUploadFileReqForTest(t, uploadBody, privateKey)
	CreatePaidAccountForTest(strings.TrimPrefix(request.Address, "0x"), t)

	fileId := request.UploadFile.FileHandle
	filesInDB := []models.File{}
	models.DB.Where("file_id = ?", fileId).Find(&filesInDB)
	assert.Equal(t, 0, len(filesInDB))

	w := UploadFileHelperForTest(t, request)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	filesInDB = []models.File{}
	models.DB.Where("file_id = ?", fileId).Find(&filesInDB)
	assert.Equal(t, 1, len(filesInDB))

	assert.NotNil(t, filesInDB[0].AwsUploadID)
	assert.NotNil(t, filesInDB[0].AwsObjectKey)

	utils.AbortMultiPartUpload(aws.StringValue(filesInDB[0].AwsObjectKey),
		aws.StringValue(filesInDB[0].AwsUploadID))
}

func Test_Upload_File_Account_Paid_Upload_Continues(t *testing.T) {
	if err := models.DB.Unscoped().Delete(&models.Account{}).Error; err != nil {
		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
	}

	uploadBody := ReturnValidUploadFileBodyForTest(t)
	uploadBody.ChunkData = string(ReturnChunkDataForTest(t))
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	request := ReturnValidUploadFileReqForTest(t, uploadBody, privateKey)
	CreatePaidAccountForTest(strings.TrimPrefix(request.Address, "0x"), t)

	w := UploadFileHelperForTest(t, request)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	nextBody := uploadBody
	nextBody.PartIndex = uploadBody.PartIndex + 1
	request = ReturnValidUploadFileReqForTest(t, nextBody, privateKey)

	w = UploadFileHelperForTest(t, request)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	filesInDB := []models.File{}
	models.DB.Where("file_id = ?", request.UploadFile.FileHandle).Find(&filesInDB)
	assert.Equal(t, 1, len(filesInDB))

	completedPartsAsArray := filesInDB[0].GetCompletedPartsAsArray()
	assert.Equal(t, 2, len(completedPartsAsArray))

	utils.AbortMultiPartUpload(aws.StringValue(filesInDB[0].AwsObjectKey),
		aws.StringValue(filesInDB[0].AwsUploadID))
}

func Test_Upload_File_Completed_File_Is_Deleted(t *testing.T) {
	uploadBody := ReturnValidUploadFileBodyForTest(t)
	uploadBody.PartIndex = models.FirstChunkIndex
	uploadBody.EndIndex = models.FirstChunkIndex + 1

	chunkData := ReturnChunkDataForTest(t)
	chunkDataPart1 := chunkData[0:utils.MaxMultiPartSizeForTest]
	chunkDataPart2 := chunkData[utils.MaxMultiPartSizeForTest:]

	uploadBody.ChunkData = string(chunkDataPart1)
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	request := ReturnValidUploadFileReqForTest(t, uploadBody, privateKey)
	CreatePaidAccountForTest(strings.TrimPrefix(request.Address, "0x"), t)

	w := UploadFileHelperForTest(t, request)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}
	uploadBody.PartIndex++
	uploadBody.ChunkData = string(chunkDataPart2)

	filesInDB := []models.File{}
	models.DB.Where("file_id = ?", request.UploadFile.FileHandle).Find(&filesInDB)
	assert.Equal(t, 1, len(filesInDB))

	objectKey := aws.StringValue(filesInDB[0].AwsObjectKey)

	request = ReturnValidUploadFileReqForTest(t, uploadBody, privateKey)

	w = UploadFileHelperForTest(t, request)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	filesInDB = []models.File{}
	models.DB.Where("file_id = ?", request.UploadFile.FileHandle).Find(&filesInDB)
	assert.Equal(t, 0, len(filesInDB))

	err = utils.DeleteDefaultBucketObject(objectKey)
	assert.Nil(t, err)
}
