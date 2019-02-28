package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bytes"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_Metadata(t *testing.T) {
	utils.SetTesting("../.env")
	gin.SetMode(gin.TestMode)
}

func Test_GetMetadataHandler_Returns_Metadata(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataKey := "testMetadataKey1"
	testMetadataValue := "testMetadataValue1"

	if err := utils.BatchSet(&utils.KVPairs{testMetadataKey: testMetadataValue}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	w := metadataTestHelperGetMetadata(t, testMetadataKey)

	// Check to see if the response was what you expected
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	assert.Contains(t, w.Body.String(), testMetadataValue)
}

func Test_GetMetadataHandler_Error_If_Not_In_KV_Store(t *testing.T) {
	testMetadataKey := "testMetadataKey2"

	w := metadataTestHelperGetMetadata(t, testMetadataKey)

	// Check to see if the response was what you expected
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusNotFound, w.Code)
	}
}

func Test_UpdateMetadataHandler_Can_Update_Metadata(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataKey := utils.RandSeqFromRunes(64, []rune("abcdef01234567890"))
	testMetadataValue := "testMetadataValue1"
	newValue := "totallyNewValue"

	if err := utils.BatchSet(&utils.KVPairs{testMetadataKey: testMetadataValue}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	post := updateMetadataReq{
		MetadataKey: testMetadataKey,
		Metadata:    newValue,
	}

	w := metadataTestHelperUpdateMetadata(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	assert.Contains(t, w.Body.String(), newValue)

	metadata, _, _ := utils.GetValueFromKV(testMetadataKey)
	assert.Equal(t, newValue, metadata)
}

func Test_UpdateMetadataHandler_Error_If_Key_Does_Not_Exist(t *testing.T) {
	testMetadataKey := utils.RandSeqFromRunes(64, []rune("abcdef01234567890"))
	newValue := "rotallyNewValue"

	post := updateMetadataReq{
		MetadataKey: testMetadataKey,
		Metadata:    newValue,
	}

	w := metadataTestHelperUpdateMetadata(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusNotFound, w.Code)
	}
}

func metadataTestHelperGetMetadata(t *testing.T, metadataKey string) *httptest.ResponseRecorder {
	router := returnEngine()
	v1 := returnV1Group(router)
	v1.GET(MetadataPath+"/:metadataKey", GetMetadataHandler())

	// Create the mock request you'd like to test. Make sure the second argument
	// here is the same as one of the routes you defined in the router setup
	// block!
	req, err := http.NewRequest(http.MethodGet, v1.BasePath()+MetadataPath+"/"+metadataKey, nil)
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
	v1.POST(MetadataPath, UpdateMetadataHandler())

	marshalledReq, _ := json.Marshal(post)
	reqBody := bytes.NewBuffer(marshalledReq)

	// Create the mock request you'd like to test. Make sure the second argument
	// here is the same as one of the routes you defined in the router setup
	// block!
	req, err := http.NewRequest(http.MethodPost, v1.BasePath()+MetadataPath, reqBody)
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
	}

	// Create a response recorder so you can inspect the response
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	return w
}