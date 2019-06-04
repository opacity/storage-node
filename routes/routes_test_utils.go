package routes

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

const defaultStorageUsedInByteForTest = 10 * 1e9

func ReturnValidUploadFileBodyForTest(t *testing.T) UploadFileObj {
	abortIfNotTesting(t)
	return UploadFileObj{
		FileHandle: utils.RandHexString(64),
		PartIndex:  models.FirstChunkIndex,
	}
}

func ReturnValidUploadFileReqForTest(t *testing.T, body UploadFileObj, privateKey *ecdsa.PrivateKey) UploadFileReq {
	abortIfNotTesting(t)

	v, b := returnValidVerificationAndRequestBody(t, body, privateKey)

	return UploadFileReq{
		verification: v,
		RequestBody:  b.RequestBody,
		ChunkData:    utils.RandHexString(64),
	}
}

func CreateUnpaidAccountForTest(t *testing.T, accountID string) models.Account {
	abortIfNotTesting(t)

	ethAddress, privateKey, _ := services.EthWrapper.GenerateWallet()

	account := models.Account{
		AccountID:            accountID,
		MonthsInSubscription: models.DefaultMonthsPerSubscription,
		StorageLocation:      "https://createdInRoutesUploadFileTest.com/12345",
		StorageLimit:         models.BasicStorageLimit,
		StorageUsedInByte:    defaultStorageUsedInByteForTest,
		PaymentStatus:        models.InitialPaymentInProgress,
		EthAddress:           ethAddress.String(),
		EthPrivateKey:        hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, accountID)),
		MetadataKey:          utils.RandHexString(64),
	}

	if err := models.DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	return account
}

func CreatePaidAccountForTest(t *testing.T, accountID string) models.Account {
	abortIfNotTesting(t)

	ethAddress, privateKey, _ := services.EthWrapper.GenerateWallet()

	account := models.Account{
		AccountID:            accountID,
		MonthsInSubscription: models.DefaultMonthsPerSubscription,
		StorageLocation:      "https://createdInRoutesUploadFileTest.com/12345",
		StorageLimit:         models.BasicStorageLimit,
		StorageUsedInByte:    defaultStorageUsedInByteForTest,
		PaymentStatus:        models.InitialPaymentReceived,
		EthAddress:           ethAddress.String(),
		EthPrivateKey:        hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, accountID)),
		MetadataKey:          utils.RandHexString(64),
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

func InitUploadFileForTest(t *testing.T, publicKey, fileID string, endIndx int) string {
	objectKey, uploadID, err := utils.CreateMultiPartUpload(fileID)
	assert.Nil(t, err)
	modifierHash, err := utils.HashString(publicKey + fileID)
	assert.Nil(t, err)
	err = models.DB.Create(&models.File{
		FileID:       fileID,
		AwsUploadID:  uploadID,
		AwsObjectKey: objectKey,
		EndIndex:     endIndx,
		ExpiredAt:    time.Now().AddDate(1, 0, 0),
		ModifierHash: modifierHash,
	}).Error
	assert.Nil(t, err)
	return *objectKey
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

func generateValidateAccountId(t *testing.T) (string, *ecdsa.PrivateKey) {
	abortIfNotTesting(t)

	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)

	publicKey := utils.PubkeyCompressedToHex(privateKey.PublicKey)
	accountId, err := utils.HashString(publicKey)
	assert.Nil(t, err)

	return accountId, privateKey
}

func returnValidVerificationAndRequestBodyWithRandomPrivateKey(t *testing.T, body interface{}) (verification, requestBody, *ecdsa.PrivateKey) {
	abortIfNotTesting(t)

	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)

	v, b := returnValidVerificationAndRequestBody(t, body, privateKey)
	return v, b, privateKey
}

func returnInvalidVerificationAndRequestBody(t *testing.T, body interface{}) (verification, requestBody, *ecdsa.PrivateKey, *ecdsa.PrivateKey) {
	abortIfNotTesting(t)

	bodyJson, _ := json.Marshal(body)
	bodyBuf := bytes.NewBuffer(bodyJson)
	b := requestBody{
		RequestBody: bodyBuf.String(),
	}

	privateKeyToSignWith, err := utils.GenerateKey()
	assert.Nil(t, err)
	wrongPrivateKey, err := utils.GenerateKey()
	assert.Nil(t, err)

	v := setupVerificationWithPrivateKeyForTest(t, bodyBuf.String(), privateKeyToSignWith)
	v.PublicKey = utils.PubkeyCompressedToHex(wrongPrivateKey.PublicKey)
	return v, b, privateKeyToSignWith, wrongPrivateKey
}

func returnValidVerificationAndRequestBody(t *testing.T, body interface{}, privateKey *ecdsa.PrivateKey) (verification, requestBody) {
	abortIfNotTesting(t)

	bodyJson, _ := json.Marshal(body)
	bodyBuf := bytes.NewBuffer(bodyJson)
	b := requestBody{
		RequestBody: bodyBuf.String(),
	}

	v := setupVerificationWithPrivateKeyForTest(t, bodyBuf.String(), privateKey)

	return v, b
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

func confirmVerifyFailedForTest(t *testing.T, w *httptest.ResponseRecorder) {
	abortIfNotTesting(t)

	// Check to see if the response was what you expected
	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusForbidden, w.Code)
	}

	assert.Contains(t, w.Body.String(), signatureDidNotMatchResponse)
}

func uploadStatusFileHelperForTest(t *testing.T, post UploadStatusReq) *httptest.ResponseRecorder {
	abortIfNotTesting(t)

	router := returnEngine()
	v1 := returnV1Group(router)
	v1.POST(UploadStatusPath, CheckUploadStatusHandler())

	marshalledReq, _ := json.Marshal(post)
	reqBody := bytes.NewBuffer(marshalledReq)

	// Create the mock request you'd like to test. Make sure the second argument
	// here is the same as one of the routes you defined in the router setup
	// block!
	req, err := http.NewRequest(http.MethodPost, v1.BasePath()+UploadStatusPath, reqBody)
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
