package routes

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"bytes"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

const defaultStorageUsedForTest = 10

func ReturnValidUploadFileBodyForTest(t *testing.T) UploadFileObj {
	abortIfNotTesting(t)
	return UploadFileObj{
		ChunkData:  utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
		FileHandle: utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
		PartIndex:  models.FirstChunkIndex,
	}
}

func ReturnValidUploadFileReqForTest(t *testing.T, body UploadFileObj, privateKey *ecdsa.PrivateKey) UploadFileReq {
	abortIfNotTesting(t)

	marshalledReq, _ := json.Marshal(body)
	reqBody := bytes.NewBuffer(marshalledReq)

	verificationBody := setupVerificationWithPrivateKeyForTest(t, reqBody.String(), privateKey)

	return UploadFileReq{
		RequestBody:  reqBody.String(),
		verification: verificationBody,
	}
}

func CreateUnpaidAccountForTest(accountID string, t *testing.T) models.Account {
	abortIfNotTesting(t)

	ethAddress, privateKey, _ := services.EthWrapper.GenerateWallet()

	account := models.Account{
		AccountID:            accountID,
		MonthsInSubscription: models.DefaultMonthsPerSubscription,
		StorageLocation:      "https://createdInRoutesUploadFileTest.com/12345",
		StorageLimit:         models.BasicStorageLimit,
		StorageUsed:          defaultStorageUsedForTest,
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

func CreatePaidAccountForTest(accountID string, t *testing.T) models.Account {
	abortIfNotTesting(t)

	ethAddress, privateKey, _ := services.EthWrapper.GenerateWallet()

	account := models.Account{
		AccountID:            accountID,
		MonthsInSubscription: models.DefaultMonthsPerSubscription,
		StorageLocation:      "https://createdInRoutesUploadFileTest.com/12345",
		StorageLimit:         models.BasicStorageLimit,
		StorageUsed:          defaultStorageUsedForTest,
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

func ReturnChunkDataForTest(t *testing.T) []byte {
	abortIfNotTesting(t)

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

func returnSuccessVerificationForTest(t *testing.T, reqBody string) verification {
	abortIfNotTesting(t)

	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	return setupVerificationWithPrivateKeyForTest(t, reqBody, privateKey)
}

func setupVerificationWithPrivateKeyForTest(t *testing.T, reqBody string, privateKey *ecdsa.PrivateKey) verification {
	abortIfNotTesting(t)

	hash := utils.Hash([]byte(reqBody))

	signature, err := utils.Sign(hash, privateKey)
	signature = signature[:utils.SigLengthInBytes]
	assert.Nil(t, err)

	verification := verification{
		Signature: hex.EncodeToString(signature),
		PublicKey: utils.PubkeyCompressedToHex(privateKey.PublicKey),
	}

	return verification
}

func returnFailedVerificationForTest(t *testing.T, reqBody string) verification {
	abortIfNotTesting(t)

	hash := utils.Hash([]byte(reqBody))

	privateKeyToSignWith, err := utils.GenerateKey()
	assert.Nil(t, err)
	wrongPrivateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	signature, err := utils.Sign(hash, privateKeyToSignWith)
	signature = signature[:utils.SigLengthInBytes]
	assert.Nil(t, err)

	verification := verification{
		Signature: hex.EncodeToString(signature),
		PublicKey: utils.PubkeyCompressedToHex(wrongPrivateKey.PublicKey),
	}

	return verification
}

func confirmVerifyFailedForTest(t *testing.T, w *httptest.ResponseRecorder) {
	abortIfNotTesting(t)

	// Check to see if the response was what you expected
	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusForbidden, w.Code)
	}

	assert.Contains(t, w.Body.String(), signatureDidNotMatchResponse)
}

func InitUploadFileForTest(t *testing.T, fileID string, endIndx int) string {
	objectKey, uploadID, err := utils.CreateMultiPartUpload(fileID)
	assert.Nil(t, err)
	err = models.DB.Create(&models.File{
		FileID:       fileID,
		AwsUploadID:  uploadID,
		AwsObjectKey: objectKey,
		EndIndex:     endIndx,
		ExpiredAt:    time.Now().AddDate(1, 0, 0),
	}).Error
	assert.Nil(t, err)
	return *objectKey
}

func FinishUploadFileForTest(t *testing.T, fileID string) (models.CompletedFile, error) {
	file, err := models.GetFileById(fileID)
	assert.Nil(t, err)
	return file.FinishUpload()
}

func UploadFileHelperForTest(t *testing.T, post UploadFileReq) *httptest.ResponseRecorder {
	abortIfNotTesting(t)

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

func UploadFileStatusHelperForTest(t *testing.T, post uploadStatusReq) *httptest.ResponseRecorder {
	abortIfNotTesting(t)

	router := returnEngine()
	v1 := returnV1Group(router)
	v1.POST(UploadStatusPath, CheckUploadStatusHandler())

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

func abortIfNotTesting(t *testing.T) {
	if !utils.IsTestEnv() {
		t.Fatalf("should only be calling this method while testing")
	}
}
