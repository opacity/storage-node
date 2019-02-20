package routes

import (
	"testing"

	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

func returnValidCreateAccountReq() accountCreateReq {
	return accountCreateReq{
		AccountID:        utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
		StorageLimit:     100,
		DurationInMonths: 12,
	}
}

func testSetupAccounts() {
	utils.SetTesting("../.env")
	models.Connect(utils.Env.DatabaseURL)
}

func Test_Init_Accounts(t *testing.T) {
	testSetupAccounts()
	gin.SetMode(gin.TestMode)
}

func Test_NoErrorsWithValidPost(t *testing.T) {
	post := returnValidCreateAccountReq()

	w := accountsTestHelper(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}
}

func Test_ExpectErrorWithInvalidAccountID(t *testing.T) {
	post := returnValidCreateAccountReq()
	post.AccountID = "abcdef"

	w := accountsTestHelper(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}
}

func Test_ExpectErrorWithInvalidStorageLimit(t *testing.T) {
	post := returnValidCreateAccountReq()
	post.StorageLimit = 99

	w := accountsTestHelper(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}
}

func Test_ExpectErrorWithInvalidDurationInMonths(t *testing.T) {
	post := returnValidCreateAccountReq()
	post.DurationInMonths = 0

	w := accountsTestHelper(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}
}

func accountsTestHelper(t *testing.T, post accountCreateReq) *httptest.ResponseRecorder {
	router := returnEngine()
	v1 := returnV1Group(router)
	v1.POST(AccountsPath, CreateAccountHandler())

	marshalledReq, _ := json.Marshal(post)
	reqBody := bytes.NewBuffer(marshalledReq)

	// Create the mock request you'd like to test. Make sure the second argument
	// here is the same as one of the routes you defined in the router setup
	// block!
	req, err := http.NewRequest(http.MethodPost, v1.BasePath()+AccountsPath, reqBody)
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
	}

	// Create a response recorder so you can inspect the response
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	return w
}
