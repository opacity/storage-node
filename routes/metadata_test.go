package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"bytes"
	"encoding/json"
	"time"

	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_Metadata(t *testing.T) {
	setupTests(t)
}

func Test_GetMetadataHandler_Returns_Metadata(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataKey := utils.RandHexString(64)
	testMetadataValue := utils.RandHexString(64)

	if err := utils.BatchSet(&utils.KVPairs{testMetadataKey: testMetadataValue}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	getMetadata := getMetadataObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getMetadata)

	get := getMetadataReq{
		verification: v,
		RequestBody:  b.RequestBody,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreatePaidAccountForTest(t, accountID)

	w := metadataTestHelperGetMetadata(t, get)

	// Check to see if the response was what you expected
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	assert.Contains(t, w.Body.String(), testMetadataValue)
}

func Test_GetMetadataHandler_Error_If_Not_Paid(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataKey := utils.RandHexString(64)
	testMetadataValue := utils.RandHexString(64)

	if err := utils.BatchSet(&utils.KVPairs{testMetadataKey: testMetadataValue}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	getMetadata := getMetadataObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getMetadata)

	get := getMetadataReq{
		verification: v,
		RequestBody:  b.RequestBody,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreateUnpaidAccountForTest(t, accountID)

	w := metadataTestHelperGetMetadata(t, get)

	// Check to see if the response was what you expected
	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusForbidden, w.Code)
	}

	assert.Contains(t, w.Body.String(), `"invoice"`)
}

func Test_GetMetadataHandler_Error_If_Not_In_KV_Store(t *testing.T) {
	testMetadataKey := utils.RandHexString(64)

	getMetadata := getMetadataObject{
		MetadataKey: testMetadataKey,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getMetadata)

	get := getMetadataReq{
		verification: v,
		RequestBody:  b.RequestBody,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreatePaidAccountForTest(t, accountID)

	w := metadataTestHelperGetMetadata(t, get)

	// Check to see if the response was what you expected
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusNotFound, w.Code)
	}
}

func Test_UpdateMetadataHandler_Can_Update_Metadata(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataKey := utils.RandHexString(64)
	testMetadataValue := utils.RandHexString(64)
	newValue := utils.RandHexString(64)

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
		RequestBody:  b.RequestBody,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreatePaidAccountForTest(t, accountID)

	w := metadataTestHelperUpdateMetadata(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	assert.Contains(t, w.Body.String(), newValue)

	metadata, _, _ := utils.GetValueFromKV(testMetadataKey)
	assert.Equal(t, newValue, metadata)
}

func Test_UpdateMetadataHandler_Error_If_Not_Paid(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataKey := utils.RandHexString(64)
	testMetadataValue := utils.RandHexString(64)
	newValue := utils.RandHexString(64)

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
		RequestBody:  b.RequestBody,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreateUnpaidAccountForTest(t, accountID)

	w := metadataTestHelperUpdateMetadata(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusForbidden, w.Code)
	}

	assert.Contains(t, w.Body.String(), `"invoice"`)
}

func Test_UpdateMetadataHandler_Error_If_Key_Does_Not_Exist(t *testing.T) {
	testMetadataKey := utils.RandHexString(64)
	newValue := utils.RandHexString(64)

	updateMetadataObj := updateMetadataObject{
		MetadataKey: testMetadataKey,
		Metadata:    newValue,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, updateMetadataObj)

	post := updateMetadataReq{
		verification: v,
		RequestBody:  b.RequestBody,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreatePaidAccountForTest(t, accountID)

	w := metadataTestHelperUpdateMetadata(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusNotFound, w.Code)
	}
}

func Test_UpdateMetadataHandler_Error_If_Verification_Fails(t *testing.T) {
	testMetadataKey := utils.RandHexString(64)
	newValue := utils.RandHexString(64)

	updateMetadataObj := updateMetadataObject{
		MetadataKey: testMetadataKey,
		Metadata:    newValue,
		Timestamp:   time.Now().Unix(),
	}

	v, b, _, _ := returnInvalidVerificationAndRequestBody(t, updateMetadataObj)

	post := updateMetadataReq{
		verification: v,
		RequestBody:  b.RequestBody,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreatePaidAccountForTest(t, accountID)

	w := metadataTestHelperUpdateMetadata(t, post)

	confirmVerifyFailedForTest(t, w)
}

func metadataTestHelperGetMetadata(t *testing.T, get getMetadataReq) *httptest.ResponseRecorder {
	router := returnEngine()
	v1 := returnV1Group(router)
	v1.GET(MetadataGetPath, GetMetadataHandler())

	marshalledReq, _ := json.Marshal(get)

	reqBody := bytes.NewBuffer(marshalledReq)

	// Create the mock request you'd like to test. Make sure the second argument
	// here is the same as one of the routes you defined in the router setup
	// block!
	req, err := http.NewRequest(http.MethodGet, v1.BasePath()+MetadataGetPath, reqBody)
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
	}

	// Create a response recorder so you can inspect the response
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	return w
}

func metadataTestHelperUpdateMetadata(t *testing.T, post updateMetadataReq) *httptest.ResponseRecorder {
	router := returnEngine()
	v1 := returnV1Group(router)
	v1.POST(MetadataSetPath, UpdateMetadataHandler())

	marshalledReq, _ := json.Marshal(post)

	reqBody := bytes.NewBuffer(marshalledReq)

	// Create the mock request you'd like to test. Make sure the second argument
	// here is the same as one of the routes you defined in the router setup
	// block!
	req, err := http.NewRequest(http.MethodPost, v1.BasePath()+MetadataSetPath, reqBody)
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
	}

	// Create a response recorder so you can inspect the response
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	return w
}
