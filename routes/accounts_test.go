package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"encoding/hex"

	"math/big"

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
		MetadataKey:      utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
	}
}

func returnValidAccount() models.Account {
	ethAddress, privateKey, _ := services.EthWrapper.GenerateWallet()

	accountID := utils.RandSeqFromRunes(64, []rune("abcdef01234567890"))

	return models.Account{
		AccountID:            accountID,
		MonthsInSubscription: models.DefaultMonthsPerSubscription,
		StorageLocation:      "https://createdInRoutesAccountsTest.com/12345",
		StorageLimit:         models.BasicStorageLimit,
		StorageUsed:          10,
		PaymentStatus:        models.InitialPaymentInProgress,
		EthAddress:           ethAddress.String(),
		EthPrivateKey:        hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, accountID)),
		MetadataKey:          utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
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

func Test_CheckAccountPaymentStatusHandler_ExpectErrorIfNoAccount(t *testing.T) {
	account := returnValidAccount()

	w := accountsTestHelperCheckAccountPaymentStatus(t, account.AccountID)

	// Check to see if the response was what you expected
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusNotFound, w.Code)
	}

	assert.Contains(t, w.Body.String(), "no account with that id")
}

func Test_CheckAccountPaymentStatusHandler_ExpectNoErrorIfAccountExists(t *testing.T) {
	account := returnValidAccount()
	// Add account to DB
	if err := models.DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return true, nil
	}

	w := accountsTestHelperCheckAccountPaymentStatus(t, account.AccountID)

	// Check to see if the response was what you expected
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	assert.Contains(t, w.Body.String(), `"paymentStatus":"paid"`)
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
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
	}

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
