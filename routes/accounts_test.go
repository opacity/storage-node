package routes

import (
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func returnValidCreateAccountBody() accountCreateObj {
	return accountCreateObj{
		StorageLimit:     int(models.BasicStorageLimit),
		DurationInMonths: 12,
	}
}

func returnValidCreateAccountReq(t *testing.T, body accountCreateObj) accountCreateReq {
	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, body)
	return accountCreateReq{
		verification: v,
		requestBody:  b,
	}
}

func returnFailedVerificationCreateAccountReq(t *testing.T, body accountCreateObj) accountCreateReq {
	v, b, _, _ := returnInvalidVerificationAndRequestBody(t, body)
	return accountCreateReq{
		verification: v,
		requestBody:  b,
	}
}

func returnValidAccountAndPrivateKey(t *testing.T) (models.Account, *ecdsa.PrivateKey) {
	accountId, privateKeyToSignWith := generateValidateAccountId(t)
	ethAddress, privateKey, _ := services.EthWrapper.GenerateWallet()

	return models.Account{
		AccountID:            accountId,
		MonthsInSubscription: models.DefaultMonthsPerSubscription,
		StorageLocation:      "https://createdInRoutesAccountsTest.com/12345",
		StorageLimit:         models.BasicStorageLimit,
		StorageUsedInByte:    10 * 1e9,
		PaymentStatus:        models.InitialPaymentInProgress,
		EthAddress:           ethAddress.String(),
		EthPrivateKey:        hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, accountId)),
	}, privateKeyToSignWith
}

func returnValidGetAccountReq(t *testing.T, body accountGetReqObj, privateKeyToSignWith *ecdsa.PrivateKey) getAccountDataReq {
	v, b := returnValidVerificationAndRequestBody(t, body, privateKeyToSignWith)

	return getAccountDataReq{
		verification: v,
		requestBody:  b,
	}
}

func Test_Init_Accounts(t *testing.T) {
	setupTests(t)
}

func Test_NoErrorsWithValidPost(t *testing.T) {
	post := returnValidCreateAccountReq(t, returnValidCreateAccountBody())

	w := httpPostRequestHelperForTest(t, AccountsPath, post)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
}

func Test_ExpectErrorWithInvalidSignature(t *testing.T) {
	post := returnValidCreateAccountReq(t, returnValidCreateAccountBody())
	post.Signature = "abcdef"

	w := httpPostRequestHelperForTest(t, AccountsPath, post)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func Test_ExpectErrorIfVerificationFails(t *testing.T) {
	post := returnFailedVerificationCreateAccountReq(t, returnValidCreateAccountBody())

	w := httpPostRequestHelperForTest(t, AccountsPath, post)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func Test_ExpectErrorWithInvalidStorageLimit(t *testing.T) {
	body := returnValidCreateAccountBody()
	body.StorageLimit = 99
	post := returnValidCreateAccountReq(t, body)

	w := httpPostRequestHelperForTest(t, AccountsPath, post)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func Test_ExpectErrorWithInvalidDurationInMonths(t *testing.T) {
	body := returnValidCreateAccountBody()
	body.DurationInMonths = 0
	post := returnValidCreateAccountReq(t, body)

	w := httpPostRequestHelperForTest(t, AccountsPath, post)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func Test_CheckAccountPaymentStatusHandler_ExpectErrorIfNoAccount(t *testing.T) {
	_, privateKey := returnValidAccountAndPrivateKey(t)
	validReq := returnValidGetAccountReq(t, accountGetReqObj{
		Timestamp: time.Now().Unix(),
	}, privateKey)

	w := httpPostRequestHelperForTest(t, AccountDataPath, validReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), noAccountWithThatID)
}

func Test_CheckAccountPaymentStatusHandler_ExpectNoErrorIfAccountExistsAndIsPaid(t *testing.T) {
	account, privateKey := returnValidAccountAndPrivateKey(t)
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

	w := httpPostRequestHelperForTest(t, AccountDataPath, validReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"paymentStatus":"paid"`)
	assert.Contains(t, w.Body.String(), `"stripePaymentExists":false`)
}

func Test_CheckAccountPaymentStatusHandler_ExpectNoErrorIfAccountExistsAndIsUnpaid(t *testing.T) {
	account, privateKey := returnValidAccountAndPrivateKey(t)
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

	w := httpPostRequestHelperForTest(t, AccountDataPath, validReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"paymentStatus":"unpaid"`)
	assert.Contains(t, w.Body.String(), `"stripePaymentExists":false`)
	assert.Contains(t, w.Body.String(), `"invoice"`)
}

func Test_CheckAccountPaymentStatusHandler_ReturnsStripeDataIfStripePaymentExists(t *testing.T) {
	models.DeleteStripePaymentsForTest(t)
	account, privateKey := returnValidAccountAndPrivateKey(t)
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

	stripeToken := services.RandTestStripeToken()
	charge, _ := services.CreateCharge(float64(utils.Env.Plans[int(account.StorageLimit)].CostInUSD), stripeToken)

	stripePayment := models.StripePayment{
		StripeToken: stripeToken,
		AccountID:   account.AccountID,
		ChargeID:    charge.ID,
	}

	models.DB.Create(&stripePayment)

	w := httpPostRequestHelperForTest(t, AccountDataPath, validReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"stripePaymentExists":true`)
	assert.Contains(t, w.Body.String(), `"paymentStatus":"paid"`)
	assert.Contains(t, w.Body.String(), stripePayment.StripeToken)

	account, _ = models.GetAccountById(account.AccountID)
	// check that from the account's perspective, it is still unpaid
	assert.Equal(t, models.InitialPaymentInProgress, account.PaymentStatus)
}
