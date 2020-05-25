package routes

import (
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_Metadata(t *testing.T) {

	setupTests(t)
}

func Test_GetMetadataHandler_Returns_Metadata(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataKey := utils.GenerateFileHandle()
	testMetadataValue := utils.GenerateFileHandle()

	if err := utils.BatchSetToDynamoKv(&utils.KVPairs{testMetadataKey: testMetadataValue}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	getMetadata := metadataKeyObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getMetadata)

	get := metadataKeyReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreatePaidAccountForTest(t, accountID)

	w := httpPostRequestHelperForTest(t, MetadataGetPath, get)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), testMetadataValue)
}

func Test_GetMetadataHandler_Error_If_Not_Paid(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataKey := utils.GenerateFileHandle()
	testMetadataValue := utils.GenerateFileHandle()

	if err := utils.BatchSetToDynamoKv(&utils.KVPairs{testMetadataKey: testMetadataValue}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	getMetadata := metadataKeyObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getMetadata)

	get := metadataKeyReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreateUnpaidAccountForTest(t, accountID)

	w := httpPostRequestHelperForTest(t, MetadataGetPath, get)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `"invoice"`)
}

func Test_GetMetadataHandler_Error_If_Not_In_KV_Store(t *testing.T) {
	testMetadataKey := utils.GenerateFileHandle()

	getMetadata := metadataKeyObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getMetadata)

	get := metadataKeyReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreatePaidAccountForTest(t, accountID)

	w := httpPostRequestHelperForTest(t, MetadataGetPath, get)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func Test_GetMetadataHistoryHandler_Returns_Metadata_History(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataKey := utils.GenerateFileHandle()
	testMetadataValue := "quick"

	getMetadataHistoryObj := metadataKeyObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getMetadataHistoryObj)

	get := metadataKeyReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreatePaidAccountForTest(t, accountID)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	permissionHash, err := getPermissionHash(v.PublicKey, testMetadataKey, c)
	assert.Nil(t, err)

	permissionHashKey := getPermissionHashKeyForBadger(testMetadataKey)

	if err := utils.BatchSetToDynamoKv(&utils.KVPairs{
		testMetadataKey: testMetadataValue,
		getVersionKeyForBadger(testMetadataKey, 0): "red",
		getVersionKeyForBadger(testMetadataKey, 1): "fox",
		getVersionKeyForBadger(testMetadataKey, 2): "jumps",
		getVersionKeyForBadger(testMetadataKey, 3): "over",
		getVersionKeyForBadger(testMetadataKey, 4): "the",
		permissionHashKey:                          permissionHash,
	}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	w := httpPostRequestHelperForTest(t, MetadataHistoryPath, get)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), testMetadataValue)
	assert.Contains(t, w.Body.String(), "metadataHistory")
	assert.Contains(t, w.Body.String(), `metadataHistory":["red","fox","jumps","over","the"]`)
}

func Test_GetMetadataHistoryHandler_Returns_Metadata_History_If_Not_Maxed_Out(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataKey := utils.GenerateFileHandle()
	testMetadataValue := "quick"

	getMetadataHistoryObj := metadataKeyObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getMetadataHistoryObj)

	get := metadataKeyReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreatePaidAccountForTest(t, accountID)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	permissionHash, err := getPermissionHash(v.PublicKey, testMetadataKey, c)
	assert.Nil(t, err)

	permissionHashKey := getPermissionHashKeyForBadger(testMetadataKey)

	if err := utils.BatchSetToDynamoKv(&utils.KVPairs{
		testMetadataKey: testMetadataValue,
		getVersionKeyForBadger(testMetadataKey, 0): "red",
		permissionHashKey:                          permissionHash,
	}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	w := httpPostRequestHelperForTest(t, MetadataHistoryPath, get)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), testMetadataValue)
	assert.Contains(t, w.Body.String(), "metadataHistory")
	assert.Contains(t, w.Body.String(), `metadataHistory":["red"]`)
}

func Test_GetMetadataHistoryHandler_Error_If_Not_Paid(t *testing.T) {
	testMetadataKey := utils.GenerateFileHandle()

	getMetadataHistoryObj := metadataKeyObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getMetadataHistoryObj)

	get := metadataKeyReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreateUnpaidAccountForTest(t, accountID)

	w := httpPostRequestHelperForTest(t, MetadataHistoryPath, get)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `"invoice"`)
}

func Test_GetMetadataHistoryHandler_Error_If_Not_In_KV_Store(t *testing.T) {
	testMetadataKey := utils.GenerateFileHandle()

	getMetadata := metadataKeyObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getMetadata)

	get := metadataKeyReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreatePaidAccountForTest(t, accountID)

	w := httpPostRequestHelperForTest(t, MetadataHistoryPath, get)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func Test_UpdateMetadataHandler_Can_Update_Metadata(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataKey := utils.GenerateFileHandle()
	testMetadataValue := utils.GenerateFileHandle()
	newValue := utils.GenerateFileHandle()

	updateMetadataObj := updateMetadataObject{
		MetadataKey: testMetadataKey,
		Metadata:    newValue,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, updateMetadataObj)

	post := updateMetadataReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)
	err := account.IncrementMetadataCount()
	assert.Nil(t, err)
	err = account.UpdateMetadataSizeInBytes(0, int64(len(testMetadataValue)))
	assert.Nil(t, err)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	permissionHash, err := getPermissionHash(v.PublicKey, testMetadataKey, c)
	assert.Nil(t, err)

	permissionHashKey := getPermissionHashKeyForBadger(testMetadataKey)

	if err := utils.BatchSetToDynamoKv(&utils.KVPairs{
		testMetadataKey:   testMetadataValue,
		permissionHashKey: permissionHash,
	}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	w := httpPostRequestHelperForTest(t, MetadataSetPath, post)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), newValue)

	metadata, _, _ := utils.GetValueFromDynamoKv(testMetadataKey)
	assert.Equal(t, newValue, metadata)

	accountFromDB, _ := models.GetAccountById(account.AccountID)
	assert.Equal(t, int64(len(newValue)), accountFromDB.TotalMetadataSizeInBytes)
}

func Test_UpdateMetadataHandler_Can_Update_Metadata_History(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataKey := utils.GenerateFileHandle()
	startingCurrentMetadataValue := "quick"
	newCurrentMetadataValue := "the"

	updateMetadataObj := updateMetadataObject{
		MetadataKey: testMetadataKey,
		Metadata:    newCurrentMetadataValue,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, updateMetadataObj)

	post := updateMetadataReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)
	err := account.IncrementMetadataCount()
	assert.Nil(t, err)
	err = account.UpdateMetadataSizeInBytes(0, int64(len(startingCurrentMetadataValue)))
	assert.Nil(t, err)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	permissionHash, err := getPermissionHash(v.PublicKey, testMetadataKey, c)
	assert.Nil(t, err)

	permissionHashKey := getPermissionHashKeyForBadger(testMetadataKey)

	if err := utils.BatchSetToDynamoKv(&utils.KVPairs{
		testMetadataKey: startingCurrentMetadataValue,
		getVersionKeyForBadger(testMetadataKey, 0): "red",
		getVersionKeyForBadger(testMetadataKey, 1): "fox",
		getVersionKeyForBadger(testMetadataKey, 2): "jumps",
		getVersionKeyForBadger(testMetadataKey, 3): "over",
		getVersionKeyForBadger(testMetadataKey, 4): "the",
		permissionHashKey:                          permissionHash,
	}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	expectedStartingMetadataHistory := []string{
		"red", "fox", "jumps", "over", "the",
	}

	expectedEndingMetadataHistory := []string{
		"quick", "red", "fox", "jumps", "over",
	}

	metadataHistory, err := getMetadataHistoryWithoutContext(testMetadataKey)
	assert.Equal(t, expectedStartingMetadataHistory, metadataHistory)
	assert.Nil(t, err)

	w := httpPostRequestHelperForTest(t, MetadataSetPath, post)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), newCurrentMetadataValue)

	metadata, _, _ := utils.GetValueFromDynamoKv(testMetadataKey)
	assert.Equal(t, newCurrentMetadataValue, metadata)

	accountFromDB, _ := models.GetAccountById(account.AccountID)
	assert.Equal(t, int64(len(newCurrentMetadataValue)), accountFromDB.TotalMetadataSizeInBytes)

	metadataHistory, err = getMetadataHistoryWithoutContext(testMetadataKey)
	assert.Equal(t, expectedEndingMetadataHistory, metadataHistory)
}

func Test_UpdateMetadataHandler_Can_Update_Metadata_History_If_Not_Maxed_Out(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataKey := utils.GenerateFileHandle()
	startingCurrentMetadataValue := "quick"
	newCurrentMetadataValue := "the"

	updateMetadataObj := updateMetadataObject{
		MetadataKey: testMetadataKey,
		Metadata:    newCurrentMetadataValue,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, updateMetadataObj)

	post := updateMetadataReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)
	err := account.IncrementMetadataCount()
	assert.Nil(t, err)
	err = account.UpdateMetadataSizeInBytes(0, int64(len(startingCurrentMetadataValue)))
	assert.Nil(t, err)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	permissionHash, err := getPermissionHash(v.PublicKey, testMetadataKey, c)
	assert.Nil(t, err)

	permissionHashKey := getPermissionHashKeyForBadger(testMetadataKey)

	if err := utils.BatchSetToDynamoKv(&utils.KVPairs{
		testMetadataKey: startingCurrentMetadataValue,
		getVersionKeyForBadger(testMetadataKey, 0): "red",
		permissionHashKey:                          permissionHash,
	}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	expectedStartingMetadataHistory := []string{
		"red",
	}

	expectedEndingMetadataHistory := []string{
		"quick", "red",
	}

	metadataHistory, err := getMetadataHistoryWithoutContext(testMetadataKey)
	assert.Equal(t, expectedStartingMetadataHistory, metadataHistory)
	assert.Nil(t, err)

	w := httpPostRequestHelperForTest(t, MetadataSetPath, post)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), newCurrentMetadataValue)

	metadata, _, _ := utils.GetValueFromDynamoKv(testMetadataKey)
	assert.Equal(t, newCurrentMetadataValue, metadata)

	accountFromDB, _ := models.GetAccountById(account.AccountID)
	assert.Equal(t, int64(len(newCurrentMetadataValue)), accountFromDB.TotalMetadataSizeInBytes)

	metadataHistory, err = getMetadataHistoryWithoutContext(testMetadataKey)
	assert.Equal(t, expectedEndingMetadataHistory, metadataHistory)
}

func Test_UpdateMetadataHandler_Error_If_Not_Paid(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataKey := utils.GenerateFileHandle()
	testMetadataValue := utils.GenerateFileHandle()
	newValue := utils.GenerateFileHandle()

	if err := utils.BatchSetToDynamoKv(&utils.KVPairs{testMetadataKey: testMetadataValue}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	updateMetadataObj := updateMetadataObject{
		MetadataKey: testMetadataKey,
		Metadata:    newValue,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, updateMetadataObj)

	post := updateMetadataReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreateUnpaidAccountForTest(t, accountID)

	w := httpPostRequestHelperForTest(t, MetadataSetPath, post)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `"invoice"`)
}

func Test_UpdateMetadataHandler_Error_If_Key_Does_Not_Exist(t *testing.T) {
	testMetadataKey := utils.GenerateFileHandle()
	newValue := utils.GenerateFileHandle()

	updateMetadataObj := updateMetadataObject{
		MetadataKey: testMetadataKey,
		Metadata:    newValue,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, updateMetadataObj)

	post := updateMetadataReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreatePaidAccountForTest(t, accountID)

	w := httpPostRequestHelperForTest(t, MetadataSetPath, post)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func Test_UpdateMetadataHandler_Error_If_Verification_Fails(t *testing.T) {
	testMetadataKey := utils.GenerateFileHandle()
	newValue := utils.GenerateFileHandle()

	updateMetadataObj := updateMetadataObject{
		MetadataKey: testMetadataKey,
		Metadata:    newValue,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _, _ := returnInvalidVerificationAndRequestBody(t, updateMetadataObj)

	post := updateMetadataReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreatePaidAccountForTest(t, accountID)

	w := httpPostRequestHelperForTest(t, MetadataSetPath, post)

	confirmVerifyFailedForTest(t, w)
}

func Test_Create_Metadata_Creates_Metadata(t *testing.T) {
	testMetadataKey := utils.RandSeqFromRunes(64, []rune("abcdef01234567890"))

	createMetadataObj := metadataKeyObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, createMetadataObj)

	post := metadataKeyReq{
		verification: v,
		requestBody: requestBody{
			RequestBody: b.RequestBody,
		},
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)
	account.StorageUsedInByte = 64 * 1e9
	err := models.DB.Save(&account).Error
	assert.Nil(t, err)

	w := httpPostRequestHelperForTest(t, MetadataCreatePath, post)

	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	permissionHashExpected, err := getPermissionHash(v.PublicKey, testMetadataKey, c)

	metadata, _, err := utils.GetValueFromDynamoKv(testMetadataKey)
	assert.Nil(t, err)
	permissionHash, _, err := utils.GetValueFromDynamoKv(getPermissionHashKeyForBadger(testMetadataKey))
	assert.Nil(t, err)
	assert.Equal(t, "", metadata)
	assert.Equal(t, permissionHashExpected, permissionHash)

	accountFromDB, _ := models.GetAccountById(account.AccountID)
	assert.Equal(t, 1, accountFromDB.TotalFolders)
}

func Test_Create_Metadata_Error_If_Unpaid_Account(t *testing.T) {
	testMetadataKey := utils.RandSeqFromRunes(64, []rune("abcdef01234567890"))

	createMetadataObj := metadataKeyObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, createMetadataObj)

	post := metadataKeyReq{
		verification: v,
		requestBody: requestBody{
			RequestBody: b.RequestBody,
		},
	}

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreateUnpaidAccountForTest(t, accountID)
	account.StorageUsedInByte = 64 * 1e9
	err := models.DB.Save(&account).Error
	assert.Nil(t, err)

	w := httpPostRequestHelperForTest(t, MetadataCreatePath, post)

	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "invoice")
}

func Test_Create_Metadata_Error_If_Too_Many_Metadatas(t *testing.T) {
	testMetadataKey := utils.RandSeqFromRunes(64, []rune("abcdef01234567890"))

	createMetadataObj := metadataKeyObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, createMetadataObj)

	post := metadataKeyReq{
		verification: v,
		requestBody: requestBody{
			RequestBody: b.RequestBody,
		},
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)
	account.TotalFolders = utils.Env.Plans[int(account.StorageLimit)].MaxFolders
	err := models.DB.Save(&account).Error
	assert.Nil(t, err)

	w := httpPostRequestHelperForTest(t, MetadataCreatePath, post)

	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusForbidden, w.Code)
	accountFromDB, _ := models.GetAccountById(account.AccountID)
	assert.Equal(t, utils.Env.Plans[int(account.StorageLimit)].MaxFolders, accountFromDB.TotalFolders)
}

func Test_Create_Metadata_Error_If_Duplicate_Metadata(t *testing.T) {
	// First create a new metadata and confirm success
	testMetadataKey := utils.RandSeqFromRunes(64, []rune("abcdef01234567890"))

	createMetadataObj := metadataKeyObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Add(-1 * time.Second).Unix(),
	}

	privateKey, _ := utils.GenerateKey()

	v, b := returnValidVerificationAndRequestBody(t, createMetadataObj, privateKey)

	post := metadataKeyReq{
		verification: v,
		requestBody: requestBody{
			RequestBody: b.RequestBody,
		},
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)
	account.StorageUsedInByte = 64 * 1e9
	err := models.DB.Save(&account).Error
	assert.Nil(t, err)

	w := httpPostRequestHelperForTest(t, MetadataCreatePath, post)

	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	accountFromDB, _ := models.GetAccountById(account.AccountID)
	assert.Equal(t, 1, accountFromDB.TotalFolders)

	// Next check that if we do another request with the same metadata key,
	// it will fail
	createMetadataObj = metadataKeyObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Unix(),
	}

	v, b = returnValidVerificationAndRequestBody(t, createMetadataObj, privateKey)

	post = metadataKeyReq{
		verification: v,
		requestBody: requestBody{
			RequestBody: b.RequestBody,
		},
	}

	w = httpPostRequestHelperForTest(t, MetadataCreatePath, post)

	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusForbidden, w.Code)

	accountFromDB, _ = models.GetAccountById(account.AccountID)
	assert.Equal(t, 1, accountFromDB.TotalFolders)
}

func Test_Delete_Metadata_Fails_If_Unpaid(t *testing.T) {
	testMetadataKey := utils.RandSeqFromRunes(64, []rune("abcdef01234567890"))

	deleteMetadataObj := metadataKeyObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Unix(),
	}

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, deleteMetadataObj)

	post := metadataKeyReq{
		verification: v,
		requestBody: requestBody{
			RequestBody: b.RequestBody,
		},
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreateUnpaidAccountForTest(t, accountID)
	account.TotalFolders = 1
	err := models.DB.Save(&account).Error
	assert.Nil(t, err)

	w := httpPostRequestHelperForTest(t, MetadataDeletePath, post)

	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "invoice")
}

func Test_Delete_Metadata_Fails_If_Permission_Hash_Does_Not_Match(t *testing.T) {
	testMetadataKey := utils.RandSeqFromRunes(64, []rune("abcdef01234567890"))
	testMetadataValue := "someValue"

	deleteMetadataObj := metadataKeyObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, deleteMetadataObj)

	post := metadataKeyReq{
		verification: v,
		requestBody: requestBody{
			RequestBody: b.RequestBody,
		},
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)
	account.TotalFolders = 1
	account.TotalMetadataSizeInBytes = int64(len(testMetadataValue))
	err := models.DB.Save(&account).Error
	assert.Nil(t, err)

	permissionHashKey := getPermissionHashKeyForBadger(testMetadataKey)

	ttl := time.Until(account.ExpirationDate())

	if err := utils.BatchSetToDynamoKv(&utils.KVPairs{
		testMetadataKey:   testMetadataValue,
		permissionHashKey: "someIncorrectPermissionHash",
	}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	w := httpPostRequestHelperForTest(t, MetadataDeletePath, post)

	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), notAuthorizedResponse)
}

func Test_Delete_Metadata_Success(t *testing.T) {
	testMetadataKey := utils.RandSeqFromRunes(64, []rune("abcdef01234567890"))
	testMetadataValue := "someValue"

	deleteMetadataObj := metadataKeyObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, deleteMetadataObj)

	post := metadataKeyReq{
		verification: v,
		requestBody: requestBody{
			RequestBody: b.RequestBody,
		},
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)
	account.TotalFolders = 1
	account.TotalMetadataSizeInBytes = int64(len(testMetadataValue))
	err := models.DB.Save(&account).Error
	assert.Nil(t, err)

	accountFromDB, _ := models.GetAccountById(account.AccountID)
	assert.Equal(t, int64(len(testMetadataValue)), accountFromDB.TotalMetadataSizeInBytes)
	assert.Equal(t, 1, accountFromDB.TotalFolders)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	permissionHash, err := getPermissionHash(v.PublicKey, testMetadataKey, c)
	permissionHashKey := getPermissionHashKeyForBadger(testMetadataKey)

	ttl := time.Until(account.ExpirationDate())

	if err := utils.BatchSetToDynamoKv(&utils.KVPairs{
		testMetadataKey:   testMetadataValue,
		permissionHashKey: permissionHash,
	}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	w := httpPostRequestHelperForTest(t, MetadataDeletePath, post)

	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), metadataDeletedRes.Status)
	accountFromDB, _ = models.GetAccountById(account.AccountID)
	assert.Equal(t, int64(0), accountFromDB.TotalMetadataSizeInBytes)
	assert.Equal(t, 0, accountFromDB.TotalFolders)
}
