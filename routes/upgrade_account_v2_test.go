package routes

import (
	"encoding/base64"
	"encoding/hex"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_UpgradeV2_Accounts(t *testing.T) {
	setupTests(t)
}

func Test_GetAccountUpgradeV2InvoiceHandler_Returns_Invoice(t *testing.T) {
	models.DeleteAccountsForTest(t)
	models.DeleteUpgradesForTest(t)

	getInvoiceObj := getUpgradeV2AccountInvoiceObject{
		StorageLimit:     2048,
		DurationInMonths: 12,
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getInvoiceObj)

	getInvoiceReq := getUpgradeV2AccountInvoiceReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)

	account.StorageLimit = models.StorageLimitType(1024)
	account.CreatedAt = time.Now().Add(time.Hour * 24 * (365 / 2) * -1)
	models.DB.Save(&account)

	w := httpPostRequestHelperForTest(t, AccountUpgradeV2InvoicePath, "v1", getInvoiceReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	//assert.Contains(t, w.Body.String(), `"usdInvoice":100`)
	assert.Contains(t, w.Body.String(), `"opctInvoice":{"cost":24,`)
}

func Test_CheckUpgradeV2StatusHandler_Returns_Status_OPCT_UpgradeV2_Success(t *testing.T) {
	models.DeleteAccountsForTest(t)
	models.DeleteUpgradesForTest(t)

	newStorageLimit := 1024

	checkUpgradeV2StatusObj := checkUpgradeV2StatusObject{
		StorageLimit:     newStorageLimit,
		DurationInMonths: models.DefaultMonthsPerSubscription,
		MetadataKeys:     []string{utils.GenerateMetadataV2Key()},
		FileHandles:      []string{utils.GenerateMetadataV2Key()},
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, checkUpgradeV2StatusObj)

	checkUpgradeV2StatusReq := checkUpgradeV2StatusReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)

	account.CreatedAt = time.Now().Add(time.Hour * 24 * (365 / 2) * -1)
	account.PaymentStatus = models.PaymentRetrievalComplete
	models.DB.Save(&account)

	CreateUpgradeV2ForTest(t, account, newStorageLimit)

	originalStorageLimit := int(account.StorageLimit)

	upgrade, err := models.GetUpgradeFromAccountIDAndStorageLimits(account.AccountID, newStorageLimit, originalStorageLimit)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)

	makeCompletedFileForTest(checkUpgradeV2StatusObj.FileHandles[0], account.ExpirationDate(), v.PublicKey)
	makeMetadataForTest(checkUpgradeV2StatusObj.MetadataKeys[0], v.PublicKey)

	completedFileStart, err := models.GetCompletedFileByFileID(checkUpgradeV2StatusObj.FileHandles[0])
	assert.Nil(t, err)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return true, nil
	}

	w := httpPostRequestHelperForTest(t, AccountUpgradeV2Path, "v1", checkUpgradeV2StatusReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	completedFileEnd, err := models.GetCompletedFileByFileID(checkUpgradeV2StatusObj.FileHandles[0])
	assert.Nil(t, err)

	account, err = models.GetAccountById(account.AccountID)
	assert.Nil(t, err)

	assert.NotEqual(t, completedFileStart.ExpiredAt, completedFileEnd.ExpiredAt)
	assert.Equal(t, completedFileEnd.ExpiredAt, account.ExpirationDate())

	assert.Equal(t, newStorageLimit, int(account.StorageLimit))
	assert.True(t, account.MonthsInSubscription > models.DefaultMonthsPerSubscription)
	assert.Contains(t, w.Body.String(), `Success with OPCT`)

	upgrade, err = models.GetUpgradeFromAccountIDAndStorageLimits(account.AccountID, newStorageLimit, originalStorageLimit)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentReceived, upgrade.PaymentStatus)
}

func Test_CheckUpgradeV2StatusHandler_Returns_Status_OPCT_UpgradeV2_Still_Pending(t *testing.T) {
	models.DeleteAccountsForTest(t)
	models.DeleteUpgradesForTest(t)
	models.DeleteStripePaymentsForTest(t)

	newStorageLimit := 1024

	checkUpgradeV2StatusObj := checkUpgradeV2StatusObject{
		StorageLimit:     newStorageLimit,
		DurationInMonths: models.DefaultMonthsPerSubscription,
		MetadataKeys:     []string{utils.GenerateMetadataV2Key()},
		FileHandles:      []string{utils.GenerateMetadataV2Key()},
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, checkUpgradeV2StatusObj)

	checkUpgradeV2StatusReq := checkUpgradeV2StatusReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)

	account.CreatedAt = time.Now().Add(time.Hour * 24 * (365 / 2) * -1)
	account.PaymentStatus = models.PaymentRetrievalComplete
	models.DB.Save(&account)

	CreateUpgradeV2ForTest(t, account, newStorageLimit)

	originalStorageLimit := int(account.StorageLimit)

	upgrade, err := models.GetUpgradeFromAccountIDAndStorageLimits(account.AccountID, newStorageLimit, originalStorageLimit)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	w := httpPostRequestHelperForTest(t, AccountUpgradeV2Path, "v1", checkUpgradeV2StatusReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	account, err = models.GetAccountById(account.AccountID)
	assert.Nil(t, err)
	assert.NotEqual(t, newStorageLimit, int(account.StorageLimit))
	assert.NotEqual(t, models.InitialPaymentReceived, account.PaymentStatus)
	assert.False(t, account.MonthsInSubscription > models.DefaultMonthsPerSubscription)
	assert.Contains(t, w.Body.String(), `Incomplete`)

	upgrade, err = models.GetUpgradeFromAccountIDAndStorageLimits(account.AccountID, newStorageLimit, originalStorageLimit)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)
}

// TODO: uncomment out if/when we support stripe for upgrade
//func Test_CheckUpgradeV2StatusHandler_Returns_Status_Stripe_UpgradeV2(t *testing.T) {
//	models.DeleteAccountsForTest(t)
//	models.DeleteUpgradesForTest(t)
//
//	newStorageLimit := 1024
//
//	checkUpgradeV2StatusObj := checkUpgradeV2StatusObject{
//		StorageLimit:     newStorageLimit,
//		DurationInMonths: models.DefaultMonthsPerSubscription,
//		MetadataKeys:     []string{utils.GenerateMetadataV2Key()},
//		FileHandles:      []string{utils.GenerateMetadataV2Key()},
//	}
//
//	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, checkUpgradeV2StatusObj)
//
//	checkUpgradeV2StatusReq := checkUpgradeV2StatusReq{
//		verification: v,
//		requestBody:  b,
//	}
//
//	accountID, _ := utils.HashString(v.PublicKey)
//	account := CreatePaidAccountForTest(t, accountID)
//
//	account.CreatedAt = time.Now().Add(time.Hour * 24 * (365 / 2) * -1)
//	account.PaymentStatus = models.PaymentRetrievalComplete
//	models.DB.Save(&account)
//
//	CreateUpgradeV2ForTest(t, account, newStorageLimit)
//
//	originalStorageLimit := int(account.StorageLimit)
//
//	upgrade, err := models.GetUpgradeV2FromAccountIDAndStorageLimits(account.AccountID, newStorageLimit, originalStorageLimit)
//	assert.Nil(t, err)
//	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)
//
//	makeCompletedFileForTest(checkUpgradeV2StatusObj.FileHandles[0], account.ExpirationDate(), v.PublicKey)
//	makeMetadataForTest(checkUpgradeV2StatusObj.MetadataKeys[0], v.PublicKey)
//
//	completedFileStart, err := models.GetCompletedFileByFileID(checkUpgradeV2StatusObj.FileHandles[0])
//	assert.Nil(t, err)
//
//	models.EthWrapper.TransferToken = func(from common.Address, privateKey *ecdsa.PrivateKey, to common.Address,
//		opctAmount big.Int, gasPrice *big.Int) (bool, string, int64) {
//		return true, "", 1
//	}
//	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
//		return false, nil
//	}
//
//	upgradeCostInUSD, _ := account.UpgradeV2CostInUSD(checkUpgradeV2StatusObj.StorageLimit,
//		checkUpgradeV2StatusObj.DurationInMonths)
//	stripeToken := services.RandTestStripeToken()
//	c, _ := gin.CreateTestContext(httptest.NewRecorder())
//	_, stripePayment, err := createChargeAndStripePayment(c, upgradeCostInUSD, account, createStripePaymentObject{
//		StripeToken:    stripeToken,
//		UpgradeV2Account: true,
//	})
//	err = payUpgradeV2CostWithStripe(c, stripePayment, account, createStripePaymentObject{
//		StorageLimit:     newStorageLimit,
//		DurationInMonths: models.DefaultMonthsPerSubscription,
//	})
//	assert.Nil(t, err)
//	account.UpdatePaymentViaStripe()
//
//	w := httpPostRequestHelperForTest(t, AccountUpgradeV2Path, checkUpgradeV2StatusReq)
//	// Check to see if the response was what you expected
//	assert.Equal(t, http.StatusOK, w.Code)
//
//	completedFileEnd, err := models.GetCompletedFileByFileID(checkUpgradeV2StatusObj.FileHandles[0])
//	assert.Nil(t, err)
//
//	account, err = models.GetAccountById(account.AccountID)
//	assert.Nil(t, err)
//
//	assert.NotEqual(t, completedFileStart.ExpiredAt, completedFileEnd.ExpiredAt)
//	assert.Equal(t, completedFileEnd.ExpiredAt, account.ExpirationDate())
//
//	assert.Equal(t, newStorageLimit, int(account.StorageLimit))
//	assert.True(t, account.MonthsInSubscription > models.DefaultMonthsPerSubscription)
//	assert.Contains(t, w.Body.String(), `Success with Stripe`)
//
//	upgrade, err = models.GetUpgradeV2FromAccountIDAndStorageLimits(account.AccountID, newStorageLimit, originalStorageLimit)
//	assert.Nil(t, err)
//	// This is in progress because we returned false for the CheckIfPaid mock
//	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)
//}

func Test_CheckUpgradeV2StatusHandler_Multiple_UpgradeV2s(t *testing.T) {
	models.DeleteAccountsForTest(t)
	models.DeleteUpgradesForTest(t)

	newStorageLimit := 1024
	newStorageLimit2 := 2048

	checkUpgradeV2StatusObj := checkUpgradeV2StatusObject{
		StorageLimit:     newStorageLimit2,
		DurationInMonths: models.DefaultMonthsPerSubscription,
		MetadataKeys:     []string{utils.GenerateMetadataV2Key()},
		FileHandles:      []string{utils.GenerateMetadataV2Key()},
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, checkUpgradeV2StatusObj)

	checkUpgradeV2StatusReq := checkUpgradeV2StatusReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)

	account.CreatedAt = time.Now().Add(time.Hour * 24 * (365 / 2) * -1)
	account.PaymentStatus = models.PaymentRetrievalComplete
	models.DB.Save(&account)

	CreateUpgradeV2ForTest(t, account, newStorageLimit)

	originalStorageLimit := int(account.StorageLimit)

	upgrade, err := models.GetUpgradeFromAccountIDAndStorageLimits(account.AccountID, newStorageLimit, originalStorageLimit)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)

	upgrade2 := returnUpgradeV2ForTest(t, account, newStorageLimit2)
	upgrade2.NewStorageLimit = models.StorageLimitType(newStorageLimit2)

	upgradeCostInOPCT, _ := account.UpgradeCostInOPCT(utils.Env.Plans[newStorageLimit2].StorageInGB,
		models.DefaultMonthsPerSubscription)
	//upgradeCostInUSD, _ := account.UpgradeV2CostInUSD(utils.Env.Plans[newStorageLimit2].StorageInGB,
	//	models.DefaultMonthsPerSubscription)

	upgrade2.OpctCost = upgradeCostInOPCT
	//upgrade2.UsdCost = upgradeCostInUSD

	err = models.DB.Create(&upgrade2).Error
	assert.Nil(t, err)

	makeCompletedFileForTest(checkUpgradeV2StatusObj.FileHandles[0], account.ExpirationDate(), v.PublicKey)
	makeMetadataForTest(checkUpgradeV2StatusObj.MetadataKeys[0], v.PublicKey)

	completedFileStart, err := models.GetCompletedFileByFileID(checkUpgradeV2StatusObj.FileHandles[0])
	assert.Nil(t, err)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return true, nil
	}

	w := httpPostRequestHelperForTest(t, AccountUpgradeV2Path, "v1", checkUpgradeV2StatusReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	completedFileEnd, err := models.GetCompletedFileByFileID(checkUpgradeV2StatusObj.FileHandles[0])
	assert.Nil(t, err)

	account, err = models.GetAccountById(account.AccountID)
	assert.Nil(t, err)

	assert.NotEqual(t, completedFileStart.ExpiredAt, completedFileEnd.ExpiredAt)
	assert.Equal(t, completedFileEnd.ExpiredAt, account.ExpirationDate())

	assert.Equal(t, newStorageLimit2, int(account.StorageLimit))
	assert.True(t, account.MonthsInSubscription > models.DefaultMonthsPerSubscription)
	assert.Contains(t, w.Body.String(), `Success with OPCT`)

	upgrade, err = models.GetUpgradeFromAccountIDAndStorageLimits(account.AccountID, newStorageLimit, originalStorageLimit)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)

	upgrade, err = models.GetUpgradeFromAccountIDAndStorageLimits(account.AccountID, newStorageLimit2, originalStorageLimit)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentReceived, upgrade.PaymentStatus)
}

func makeCompletedFileV2ForTest(handle string, expirationDate time.Time, key string) {
	completedFile := models.CompletedFile{
		FileID:         handle,
		FileSizeInByte: 150,
		ExpiredAt:      expirationDate,
	}
	modifierHash, _ := utils.HashString(key + completedFile.FileID)
	completedFile.ModifierHash = modifierHash
	models.DB.Save(&completedFile)
}

func makeMetadataV2ForTest(metadataKey string, key string) {
	ttl := utils.TestValueTimeToLive

	keyBytes, _ := base64.URLEncoding.DecodeString(key)
	metadataKeyBytes, _ := base64.URLEncoding.DecodeString(metadataKey)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	permissionHash := getPermissionHashV2(keyBytes, metadataKeyBytes, c)

	permissionHashKey := getPermissionHashV2KeyForBadger(metadataKey)
	utils.BatchSet(&utils.KVPairs{
		metadataKey:       "",
		permissionHashKey: permissionHash,
	}, ttl)
}

func CreateUpgradeV2ForTest(t *testing.T, account models.Account, newStorageLimit int) models.Upgrade {
	upgrade := returnUpgradeV2ForTest(t, account, newStorageLimit)

	if err := models.DB.Create(&upgrade).Error; err != nil {
		t.Fatalf("should have created upgrade but didn't: " + err.Error())
	}

	return upgrade
}

func returnUpgradeV2ForTest(t *testing.T, account models.Account, newStorageLimit int) models.Upgrade {
	abortIfNotTesting(t)

	ethAddress, privateKey, _ := services.EthWrapper.GenerateWallet()

	upgradeCostInOPCT, _ := account.UpgradeCostInOPCT(utils.Env.Plans[newStorageLimit].StorageInGB,
		models.DefaultMonthsPerSubscription)
	//upgradeCostInUSD, _ := account.UpgradeV2CostInUSD(utils.Env.Plans[newStorageLimit].StorageInGB,
	//	models.DefaultMonthsPerSubscription)

	return models.Upgrade{
		AccountID:        account.AccountID,
		NewStorageLimit:  models.StorageLimitType(newStorageLimit),
		OldStorageLimit:  account.StorageLimit,
		DurationInMonths: models.DefaultMonthsPerSubscription,
		PaymentStatus:    models.InitialPaymentInProgress,
		OpctCost:         upgradeCostInOPCT,
		//UsdCost:          upgradeCostInUSD,
		EthAddress:    ethAddress.String(),
		EthPrivateKey: hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, account.AccountID)),
	}
}
