package routes

import (
	"net/http"
	"testing"
	"time"

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

	if err := utils.BatchSet(&utils.KVPairs{testMetadataKey: testMetadataValue}, ttl); err != nil {
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

	if err := utils.BatchSet(&utils.KVPairs{testMetadataKey: testMetadataValue}, ttl); err != nil {
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

func Test_UpdateMetadataHandler_Can_Update_Metadata(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataKey := utils.GenerateFileHandle()
	testMetadataValue := utils.GenerateFileHandle()
	newValue := utils.GenerateFileHandle()

	if err := utils.BatchSet(&utils.KVPairs{testMetadataKey: testMetadataValue}, ttl); err != nil {
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
	CreatePaidAccountForTest(t, accountID)

	w := httpPostRequestHelperForTest(t, MetadataSetPath, post)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), newValue)

	metadata, _, _ := utils.GetValueFromKV(testMetadataKey)
	assert.Equal(t, newValue, metadata)
}

func Test_UpdateMetadataHandler_Error_If_Not_Paid(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataKey := utils.GenerateFileHandle()
	testMetadataValue := utils.GenerateFileHandle()
	newValue := utils.GenerateFileHandle()

	if err := utils.BatchSet(&utils.KVPairs{testMetadataKey: testMetadataValue}, ttl); err != nil {
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
