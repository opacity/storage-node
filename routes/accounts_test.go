package routes

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func returnValidCreateAccountReq() accountCreateReq {
	return accountCreateReq{
		AccountID:        utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
		StorageLimit:     100,
		DurationInMonths: 12,
	}
}

func returnValidAccount() models.Account {
	ethAddress, privateKey, _ := services.EthWrapper.GenerateWallet()

	accountID := utils.RandSeqFromRunes(64, []rune("abcdef01234567890"))

	return models.Account{
		AccountID:            accountID,
		MonthsInSubscription: 12,
		StorageLocation:      "https://someFileStoragePlace.com/12345",
		StorageLimit:         models.BasicStorageLimit,
		StorageUsed:          10,
		PaymentStatus:        models.InitialPaymentInProgress,
		EthAddress:           ethAddress.String(),
		EthPrivateKey:        hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, accountID)),
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

	w := accountsTestHelperCreateAccount(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}
}

func Test_ExpectErrorWithInvalidAccountID(t *testing.T) {
	post := returnValidCreateAccountReq()
	post.AccountID = "abcdef"

	w := accountsTestHelperCreateAccount(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}
}

func Test_ExpectErrorWithInvalidStorageLimit(t *testing.T) {
	post := returnValidCreateAccountReq()
	post.StorageLimit = 99

	w := accountsTestHelperCreateAccount(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}
}

func Test_ExpectErrorWithInvalidDurationInMonths(t *testing.T) {
	post := returnValidCreateAccountReq()
	post.DurationInMonths = 0

	w := accountsTestHelperCreateAccount(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}
}

func Test_ExpectErrorIfNoAccount(t *testing.T) {
	account := returnValidAccount()

	w := accountsTestHelperCheckAccountPaymentStatus(t, account.AccountID)

	// Check to see if the response was what you expected
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusNotFound, w.Code)
	}

	assert.Contains(t, w.Body.String(), "no account with that id")
}

func Test_ExpectNoErrorIfAccountExists(t *testing.T) {
	account := returnValidAccount()
	// Add account to DB
	assert.Nil(t, models.DB.Create(&account).Error)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return true, nil
	}

	w := accountsTestHelperCheckAccountPaymentStatus(t, account.AccountID)

	// Check to see if the response was what you expected
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	assert.Contains(t, w.Body.String(), `"paid":true`)
}

func Test_HasEnoughSpaceToUploadFile(t *testing.T) {
	account := returnValidAccount()
	account.PaymentStatus = models.PaymentRetrievalComplete
	assert.Nil(t, models.DB.Create(&account).Error)

	assert.Nil(t, account.UpdateStorageUsedInByte(10*1e9 /* Upload 10GB. */))
}

func Test_NoEnoughSpaceToUploadFile(t *testing.T) {
	// account := returnValidAccount()
	// account.PaymentStatus = models.PaymentRetrievalComplete
	// assert.Nil(t, models.DB.Create(&account).Error)

	// assert.NotNil(t, account.UpdateStorageUsedInByte(95*1e9 /* Upload 95GB. */))
}

func accountsTestHelperCreateAccount(t *testing.T, post accountCreateReq) *httptest.ResponseRecorder {
	router := returnEngine()
	v1 := returnV1Group(router)
	v1.POST(AccountsPath, CreateAccountHandler())

	marshalledReq, _ := json.Marshal(post)
	reqBody := bytes.NewBuffer(marshalledReq)

	// Create the mock request you'd like to test. Make sure the second argument
	// here is the same as one of the routes you defined in the router setup
	// block!
	req, err := http.NewRequest(http.MethodPost, v1.BasePath()+AccountsPath, reqBody)
	assert.Nil(t, err, fmt.Sprintf("Couldn't create request %v", err))

	// Create a response recorder so you can inspect the response
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	return w
}

func accountsTestHelperCheckAccountPaymentStatus(t *testing.T, accountID string) *httptest.ResponseRecorder {
	router := returnEngine()
	v1 := returnV1Group(router)
	v1.GET(AccountsPath+"/:accountID", CheckAccountPaymentStatusHandler())

	// Create the mock request you'd like to test. Make sure the second argument
	// here is the same as one of the routes you defined in the router setup
	// block!
	req, err := http.NewRequest(http.MethodGet, v1.BasePath()+AccountsPath+"/"+accountID, nil)
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
	}

	// Create a response recorder so you can inspect the response
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	return w
}
