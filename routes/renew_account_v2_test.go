package routes

import (
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

func Test_Init_RenewV2_Accounts(t *testing.T) {
	setupTests(t)
}

func Test_GetAccountRenewV2InvoiceHandler_Returns_Invoice(t *testing.T) {
	models.DeleteAccountsForTest(t)
	models.DeleteRenewalsForTest(t)

	getInvoiceObj := getRenewalV2AccountInvoiceObject{}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getInvoiceObj)

	getInvoiceReq := getRenewalV2AccountInvoiceReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)

	account.StorageLimit = models.StorageLimitType(1024)
	account.CreatedAt = time.Now().Add(time.Hour * 24 * 360 * -1)
	models.DB.Save(&account)

	w := httpPostRequestHelperForTest(t, AccountRenewInvoicePath, "v1", getInvoiceReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	// TODO: uncomment out if we decide to support stripe for renewals
	// assert.Contains(t, w.Body.String(), `"usdInvoice":100`)
	assert.Contains(t, w.Body.String(), `"opctInvoice":{"cost":16,`)
}

func Test_GetAccountRenewV2InvoiceHandler_ReturnsErrorIfExpirationDateTooFarInFuture(t *testing.T) {
	models.DeleteAccountsForTest(t)
	models.DeleteRenewalsForTest(t)

	getInvoiceObj := getRenewalV2AccountInvoiceObject{}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getInvoiceObj)

	getInvoiceReq := getRenewalV2AccountInvoiceReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)

	account.StorageLimit = models.StorageLimitType(1024)
	account.MonthsInSubscription = 13
	models.DB.Save(&account)

	w := httpPostRequestHelperForTest(t, AccountRenewInvoicePath, "v1", getInvoiceReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `account has too much time left to renew`)
}

func Test_CheckRenewalV2StatusHandler_Returns_Status_OPCT_Renew_Success(t *testing.T) {
	t.SkipNow()

	models.DeleteAccountsForTest(t)
	models.DeleteRenewalsForTest(t)

	checkRenewalV2StatusObj := checkRenewalV2StatusObject{
		MetadataKeys: []string{utils.GenerateMetadataV2Key()},
		FileHandles:  []string{utils.GenerateMetadataV2Key()},
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, checkRenewalV2StatusObj)

	checkRenewalV2StatusReq := checkRenewalV2StatusReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)

	account.CreatedAt = time.Now().Add(time.Hour * 24 * 360 * -1)
	account.PaymentStatus = models.PaymentRetrievalComplete
	models.DB.Save(&account)

	originalMonthsInSubscription := account.MonthsInSubscription

	CreateRenewalV2ForTest(t, account)

	renewals, err := models.GetRenewalsFromAccountID(account.AccountID)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, renewals[0].PaymentStatus)

	makeCompletedFileForTest(checkRenewalV2StatusObj.FileHandles[0], account.ExpirationDate(), v.PublicKey)
	makeMetadataForTest(checkRenewalV2StatusObj.MetadataKeys[0], v.PublicKey)

	completedFileStart, err := models.GetCompletedFileByFileID(checkRenewalV2StatusObj.FileHandles[0])
	assert.Nil(t, err)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return true, utils.TestNetworkID, nil
	}

	w := httpPostRequestHelperForTest(t, AccountRenewPath, "v1", checkRenewalV2StatusReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	completedFileEnd, err := models.GetCompletedFileByFileID(checkRenewalV2StatusObj.FileHandles[0])
	assert.Nil(t, err)

	account, err = models.GetAccountById(account.AccountID)
	assert.Nil(t, err)

	assert.NotEqual(t, completedFileStart.ExpiredAt, completedFileEnd.ExpiredAt)
	assert.Equal(t, completedFileEnd.ExpiredAt, account.ExpirationDate())

	assert.Equal(t, originalMonthsInSubscription+12, account.MonthsInSubscription)
	assert.True(t, account.MonthsInSubscription > models.DefaultMonthsPerSubscription)
	assert.Contains(t, w.Body.String(), `Success with OPCT`)

	renewals, err = models.GetRenewalsFromAccountID(account.AccountID)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentReceived, renewals[0].PaymentStatus)
}

func Test_CheckRenewalV2StatusHandler_Returns_Status_OPCT_Renew_Still_Pending(t *testing.T) {
	models.DeleteAccountsForTest(t)
	models.DeleteRenewalsForTest(t)
	models.DeleteStripePaymentsForTest(t)

	checkRenewalV2StatusObj := checkRenewalV2StatusObject{
		MetadataKeys: []string{utils.GenerateMetadataV2Key()},
		FileHandles:  []string{utils.GenerateMetadataV2Key()},
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, checkRenewalV2StatusObj)

	checkRenewalV2StatusReq := checkRenewalV2StatusReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)

	account.CreatedAt = time.Now().Add(time.Hour * 24 * 360 * -1)
	account.PaymentStatus = models.PaymentRetrievalComplete
	models.DB.Save(&account)

	originalMonthsInSubscription := account.MonthsInSubscription

	CreateRenewalV2ForTest(t, account)

	renewals, err := models.GetRenewalsFromAccountID(account.AccountID)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, renewals[0].PaymentStatus)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return false, 0, nil
	}

	w := httpPostRequestHelperForTest(t, AccountRenewPath, "v1", checkRenewalV2StatusReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	account, err = models.GetAccountById(account.AccountID)
	assert.Nil(t, err)
	assert.Equal(t, originalMonthsInSubscription, account.MonthsInSubscription)
	assert.NotEqual(t, models.InitialPaymentReceived, account.PaymentStatus)
	assert.False(t, account.MonthsInSubscription > models.DefaultMonthsPerSubscription)
	assert.Contains(t, w.Body.String(), `Incomplete`)

	renewals, err = models.GetRenewalsFromAccountID(account.AccountID)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, renewals[0].PaymentStatus)
}

func CreateRenewalV2ForTest(t *testing.T, account models.Account) models.Renewal {
	renewal := returnRenewalV2ForTest(t, account)

	if err := models.DB.Create(&renewal).Error; err != nil {
		t.Fatalf("should have created renewal but didn't: " + err.Error())
	}

	return renewal
}

func returnRenewalV2ForTest(t *testing.T, account models.Account) models.Renewal {
	abortIfNotTesting(t)

	ethAddress, privateKey := services.GenerateWallet()

	renewalCostInOPCT, _ := account.Cost()

	return models.Renewal{
		AccountID:        account.AccountID,
		DurationInMonths: models.DefaultMonthsPerSubscription,
		PaymentStatus:    models.InitialPaymentInProgress,
		OpctCost:         renewalCostInOPCT,
		EthAddress:       ethAddress.String(),
		EthPrivateKey:    hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, account.AccountID)),
	}
}
