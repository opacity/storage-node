package routes

import (
	"encoding/hex"
	"testing"

	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"os"
	"strings"

	"math/big"

	"crypto/ecdsa"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func returnValidUploadFileBody() uploadFileObj {
	return uploadFileObj{
		ChunkData:  utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
		FileHandle: utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
		PartIndex:  models.FirstChunkIndex,
		EndIndex:   10,
	}
}

func returnValidUploadFileReq(t *testing.T, body uploadFileObj, privateKey *ecdsa.PrivateKey) uploadFileReq {
	verificationBody := setupVerificationWithPrivateKey(t, body, privateKey)

	return uploadFileReq{
		UploadFile:   body,
		verification: verificationBody,
	}
}

func createUnpaidAccount(accountID string, t *testing.T) models.Account {
	ethAddress, privateKey, _ := services.EthWrapper.GenerateWallet()

	account := models.Account{
		AccountID:            accountID,
		MonthsInSubscription: models.DefaultMonthsPerSubscription,
		StorageLocation:      "https://createdInRoutesUploadFileTest.com/12345",
		StorageLimit:         models.BasicStorageLimit,
		StorageUsed:          10,
		PaymentStatus:        models.InitialPaymentInProgress,
		EthAddress:           ethAddress.String(),
		EthPrivateKey:        hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, accountID)),
		MetadataKey:          utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
	}

	if err := models.DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	return account
}

func createPaidAccount(accountID string, t *testing.T) models.Account {
	ethAddress, privateKey, _ := services.EthWrapper.GenerateWallet()

	account := models.Account{
		AccountID:            accountID,
		MonthsInSubscription: models.DefaultMonthsPerSubscription,
		StorageLocation:      "https://createdInRoutesUploadFileTest.com/12345",
		StorageLimit:         models.BasicStorageLimit,
		StorageUsed:          10,
		PaymentStatus:        models.InitialPaymentReceived,
		EthAddress:           ethAddress.String(),
		EthPrivateKey:        hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, accountID)),
		MetadataKey:          utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
	}

	if err := models.DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	return account
}

func returnChunkData(t *testing.T) []byte {
	workingDir, _ := os.Getwd()
	testDir := strings.Replace(workingDir, "/routes", "", -1)
	testDir = testDir + "/test_files"
	localFilePath := testDir + string(os.PathSeparator) + "lorem.txt"

	file, err := os.Open(localFilePath)
	assert.Nil(t, err)
	defer file.Close()
	fileInfo, _ := file.Stat()
	size := fileInfo.Size()
	buffer := make([]byte, size)
	file.Read(buffer)

	return buffer
}

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
	request := returnValidUploadFileReq(t, returnValidUploadFileBody(), privateKey)
	request.UploadFile.PartIndex = 0

	w := uploadFileHelper(t, request)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}
}

func Test_Upload_File_No_Account_Found(t *testing.T) {
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	request := returnValidUploadFileReq(t, returnValidUploadFileBody(), privateKey)

	w := uploadFileHelper(t, request)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusNotFound, w.Code)
	}
}

func Test_Upload_File_Account_Not_Paid(t *testing.T) {
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	request := returnValidUploadFileReq(t, returnValidUploadFileBody(), privateKey)
	createUnpaidAccount(strings.TrimPrefix(request.Address, "0x"), t)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	w := uploadFileHelper(t, request)

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

	uploadBody := returnValidUploadFileBody()
	uploadBody.ChunkData = string(returnChunkData(t))
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	request := returnValidUploadFileReq(t, uploadBody, privateKey)
	createPaidAccount(strings.TrimPrefix(request.Address, "0x"), t)

	fileId := request.UploadFile.FileHandle
	filesInDB := []models.File{}
	models.DB.Where("file_id = ?", fileId).Find(&filesInDB)
	assert.Equal(t, 0, len(filesInDB))

	w := uploadFileHelper(t, request)

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

	uploadBody := returnValidUploadFileBody()
	uploadBody.ChunkData = string(returnChunkData(t))
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	request := returnValidUploadFileReq(t, uploadBody, privateKey)
	createPaidAccount(strings.TrimPrefix(request.Address, "0x"), t)

	w := uploadFileHelper(t, request)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	nextBody := uploadBody
	nextBody.PartIndex = uploadBody.PartIndex + 1
	request = returnValidUploadFileReq(t, nextBody, privateKey)

	w = uploadFileHelper(t, request)

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
	uploadBody := returnValidUploadFileBody()
	uploadBody.PartIndex = models.FirstChunkIndex
	uploadBody.EndIndex = models.FirstChunkIndex + 1

	chunkData := returnChunkData(t)
	chunkDataPart1 := chunkData[0:utils.MaxMultiPartSizeForTest]
	chunkDataPart2 := chunkData[utils.MaxMultiPartSizeForTest:]

	uploadBody.ChunkData = string(chunkDataPart1)
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	request := returnValidUploadFileReq(t, uploadBody, privateKey)
	createPaidAccount(strings.TrimPrefix(request.Address, "0x"), t)

	w := uploadFileHelper(t, request)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}
	uploadBody.PartIndex++
	uploadBody.ChunkData = string(chunkDataPart2)

	filesInDB := []models.File{}
	models.DB.Where("file_id = ?", request.UploadFile.FileHandle).Find(&filesInDB)
	assert.Equal(t, 1, len(filesInDB))

	objectKey := aws.StringValue(filesInDB[0].AwsObjectKey)

	request = returnValidUploadFileReq(t, uploadBody, privateKey)

	w = uploadFileHelper(t, request)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	filesInDB = []models.File{}
	models.DB.Where("file_id = ?", request.UploadFile.FileHandle).Find(&filesInDB)
	assert.Equal(t, 0, len(filesInDB))

	err = utils.DeleteDefaultBucketObject(objectKey)
	assert.Nil(t, err)
}

func uploadFileHelper(t *testing.T, post uploadFileReq) *httptest.ResponseRecorder {
	router := returnEngine()
	v1 := returnV1Group(router)
	v1.POST(UploadPath, UploadFileHandler())

	marshalledReq, _ := json.Marshal(post)
	reqBody := bytes.NewBuffer(marshalledReq)

	// Create the mock request you'd like to test. Make sure the second argument
	// here is the same as one of the routes you defined in the router setup
	// block!
	req, err := http.NewRequest(http.MethodPost, v1.BasePath()+UploadPath, reqBody)
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
	}

	// Create a response recorder so you can inspect the response
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	return w
}
