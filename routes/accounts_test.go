package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"encoding/hex"

	"math/big"

	"crypto/ecdsa"

	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func returnValidCreateAccountBody() accountCreateObj {
	return accountCreateObj{
		StorageLimit:     int(models.BasicStorageLimit),
		DurationInMonths: 12,
		MetadataKey:      utils.RandHexString(64),
	}
}

func returnValidCreateAccountReq(t *testing.T, body accountCreateObj) accountCreateReq {
	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, body)

	return accountCreateReq{
		verification: v,
		RequestBody:  b.RequestBody,
	}
}

func returnFailedVerificationCreateAccountReq(t *testing.T, body accountCreateObj) accountCreateReq {
	v, b, _, _ := returnInvalidVerificationAndRequestBody(t, body)

	return accountCreateReq{
		verification: v,
		RequestBody:  b.RequestBody,
	}
}

func returnValidAccountAndPrivateKey() (models.Account, *ecdsa.PrivateKey) {
	privateKeyToSignWith, _ := utils.GenerateKey()

	accountID, _ := utils.HashString(utils.PubkeyCompressedToHex(privateKeyToSignWith.PublicKey))

	ethAddress, privateKey, _ := services.EthWrapper.GenerateWallet()

	return models.Account{
		AccountID:            accountID,
		MonthsInSubscription: models.DefaultMonthsPerSubscription,
		StorageLocation:      "https://createdInRoutesAccountsTest.com/12345",
		StorageLimit:         models.BasicStorageLimit,
		StorageUsed:          10,
		PaymentStatus:        models.InitialPaymentInProgress,
		EthAddress:           ethAddress.String(),
		EthPrivateKey:        hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, accountID)),
		MetadataKey:          utils.RandHexString(64),
	}, privateKeyToSignWith
}

func returnValidGetAccountReq(t *testing.T, body accountGetReqObj, privateKeyToSignWith *ecdsa.PrivateKey) getAccountDataReq {
	v, b := returnValidVerificationAndRequestBody(t, body, privateKeyToSignWith)

	return getAccountDataReq{
		verification: v,
		RequestBody:  b.RequestBody,
	}
}

func returnValidAccount() models.Account {
	ethAddress, privateKey, _ := services.EthWrapper.GenerateWallet()

	accountID := utils.RandSeqFromRunes(models.AccountIDLength, []rune("abcdef01234567890"))

	return models.Account{
		AccountID:            accountID,
		MonthsInSubscription: models.DefaultMonthsPerSubscription,
		StorageLocation:      "https://createdInRoutesAccountsTest.com/12345",
		StorageLimit:         models.BasicStorageLimit,
		StorageUsed:          10,
		PaymentStatus:        models.InitialPaymentInProgress,
		EthAddress:           ethAddress.String(),
		EthPrivateKey:        hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, accountID)),
		MetadataKey:          utils.RandHexString(64),
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
	post := returnValidCreateAccountReq(t, returnValidCreateAccountBody())

	w := accountsTestHelperCreateAccount(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}
}

func Test_ExpectErrorWithInvalidSignature(t *testing.T) {
	post := returnValidCreateAccountReq(t, returnValidCreateAccountBody())
	post.Signature = "abcdef"

	w := accountsTestHelperCreateAccount(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}
}

func Test_ExpectErrorIfVerificationFails(t *testing.T) {
	post := returnFailedVerificationCreateAccountReq(t, returnValidCreateAccountBody())

	w := accountsTestHelperCreateAccount(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusForbidden, w.Code)
	}
}

func Test_ExpectErrorWithInvalidStorageLimit(t *testing.T) {
	body := returnValidCreateAccountBody()
	body.StorageLimit = 99
	post := returnValidCreateAccountReq(t, body)

	w := accountsTestHelperCreateAccount(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}
}

func Test_ExpectErrorWithInvalidDurationInMonths(t *testing.T) {
	body := returnValidCreateAccountBody()
	body.DurationInMonths = 0
	post := returnValidCreateAccountReq(t, body)

	w := accountsTestHelperCreateAccount(t, post)

	// Check to see if the response was what you expected
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}
}

func Test_CheckAccountPaymentStatusHandler_ExpectErrorIfNoAccount(t *testing.T) {
	_, privateKey := returnValidAccountAndPrivateKey()
	validReq := returnValidGetAccountReq(t, accountGetReqObj{
		Timestamp: time.Now().Unix(),
	}, privateKey)

	w := accountsTestHelperCheckAccountPaymentStatus(t, validReq)

	// Check to see if the response was what you expected
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusNotFound, w.Code)
	}

	assert.Contains(t, w.Body.String(), "no account with that id")
}

func Test_CheckAccountPaymentStatusHandler_ExpectNoErrorIfAccountExistsAndIsPaid(t *testing.T) {
	account, privateKey := returnValidAccountAndPrivateKey()
	validReq := returnValidGetAccountReq(t, accountGetReqObj{
		Timestamp: time.Now().Unix(),
	}, privateKey)
	//	// Add account to DB
	if err := models.DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return true, nil
	}

	w := accountsTestHelperCheckAccountPaymentStatus(t, validReq)

	// Check to see if the response was what you expected
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	assert.Contains(t, w.Body.String(), `"paymentStatus":"paid"`)
}

func Test_CheckAccountPaymentStatusHandler_ExpectNoErrorIfAccountExistsAndIsUnpaid(t *testing.T) {
	account, privateKey := returnValidAccountAndPrivateKey()
	validReq := returnValidGetAccountReq(t, accountGetReqObj{
		Timestamp: time.Now().Unix(),
	}, privateKey)
	//	// Add account to DB
	if err := models.DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	w := accountsTestHelperCheckAccountPaymentStatus(t, validReq)

	// Check to see if the response was what you expected
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	assert.Contains(t, w.Body.String(), `"paymentStatus":"unpaid"`)
	assert.Contains(t, w.Body.String(), `"invoice"`)
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

func accountsTestHelperCheckAccountPaymentStatus(t *testing.T, get getAccountDataReq) *httptest.ResponseRecorder {
	router := returnEngine()
	v1 := returnV1Group(router)
	v1.POST(AccountDataPath, CheckAccountPaymentStatusHandler())

	marshalledReq, _ := json.Marshal(get)
	reqBody := bytes.NewBuffer(marshalledReq)

	// Create the mock request you'd like to test. Make sure the second argument
	// here is the same as one of the routes you defined in the router setup
	// block!
	req, err := http.NewRequest(http.MethodPost, v1.BasePath()+AccountDataPath, reqBody)
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
	}

	// Create a response recorder so you can inspect the response
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	return w
}
