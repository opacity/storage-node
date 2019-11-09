package routes

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Test_Init_Upgrade_Accounts(t *testing.T) {
	setupTests(t)
}

func Test_GetAccountUpgradeInvoiceHandler_Returns_Invoice(t *testing.T) {
	getInvoiceObj := getUpgradeAccountInvoiceObject{
		StorageLimit:     2048,
		DurationInMonths: 12,
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getInvoiceObj)

	getInvoiceReq := getUpgradeAccountInvoiceReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)

	account.StorageLimit = models.StorageLimitType(1024)
	account.CreatedAt = time.Now().Add(time.Hour * 24 * (365 / 2) * -1)
	models.DB.Save(&account)

	w := httpPostRequestHelperForTest(t, AccountUpgradeInvoicePath, getInvoiceReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"usdInvoice":100`)
	assert.Contains(t, w.Body.String(), `"opqInvoice":{"cost":24,`)
}

func Test_CheckUpgradeStatusHandler_Returns_Status_OPQ_Upgrade_Success(t *testing.T) {
	newStorageLimit := 2048

	checkUpgradeStatusObj := checkUpgradeStatusObject{
		StorageLimit:     newStorageLimit,
		DurationInMonths: models.DefaultMonthsPerSubscription,
		MetadataKeys:     []string{utils.GenerateFileHandle()},
		FileHandles:      []string{utils.GenerateFileHandle()},
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, checkUpgradeStatusObj)

	checkUpgradeStatusReq := checkUpgradeStatusReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)

	account.CreatedAt = time.Now().Add(time.Hour * 24 * (365 / 2) * -1)
	account.PaymentStatus = models.PaymentRetrievalComplete
	models.DB.Save(&account)

	makeCompletedFileForTest(checkUpgradeStatusObj.FileHandles[0], account.ExpirationDate(), v.PublicKey)
	makeMetadataForTest(checkUpgradeStatusObj.MetadataKeys[0], v.PublicKey)

	completedFileStart, err := models.GetCompletedFileByFileID(checkUpgradeStatusObj.FileHandles[0])
	assert.Nil(t, err)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return true, nil
	}

	w := httpPostRequestHelperForTest(t, AccountUpgradePath, checkUpgradeStatusReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	completedFileEnd, err := models.GetCompletedFileByFileID(checkUpgradeStatusObj.FileHandles[0])
	assert.Nil(t, err)

	account, err = models.GetAccountById(account.AccountID)
	assert.Nil(t, err)

	assert.NotEqual(t, completedFileStart.ExpiredAt, completedFileEnd.ExpiredAt)
	assert.Equal(t, completedFileEnd.ExpiredAt, account.ExpirationDate())

	assert.Equal(t, newStorageLimit, int(account.StorageLimit))
	assert.Equal(t, models.InitialPaymentReceived, account.PaymentStatus)
	assert.True(t, account.MonthsInSubscription > models.DefaultMonthsPerSubscription)
	assert.Contains(t, w.Body.String(), `Success with OPQ`)
}

func Test_CheckUpgradeStatusHandler_Returns_Status_OPQ_Upgrade_Still_Pending(t *testing.T) {
	models.DeleteStripePaymentsForTest(t)
	newStorageLimit := 2048
	checkUpgradeStatusObj := checkUpgradeStatusObject{
		StorageLimit:     newStorageLimit,
		DurationInMonths: models.DefaultMonthsPerSubscription,
		MetadataKeys:     []string{utils.GenerateFileHandle()},
		FileHandles:      []string{utils.GenerateFileHandle()},
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, checkUpgradeStatusObj)

	checkUpgradeStatusReq := checkUpgradeStatusReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)

	account.CreatedAt = time.Now().Add(time.Hour * 24 * (365 / 2) * -1)
	account.PaymentStatus = models.PaymentRetrievalComplete
	models.DB.Save(&account)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	w := httpPostRequestHelperForTest(t, AccountUpgradePath, checkUpgradeStatusReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	account, err := models.GetAccountById(account.AccountID)
	assert.Nil(t, err)
	assert.NotEqual(t, newStorageLimit, int(account.StorageLimit))
	assert.NotEqual(t, models.InitialPaymentReceived, account.PaymentStatus)
	assert.False(t, account.MonthsInSubscription > models.DefaultMonthsPerSubscription)
	assert.Contains(t, w.Body.String(), `Incomplete`)
}

func Test_CheckUpgradeStatusHandler_Returns_Status_Stripe_Upgrade(t *testing.T) {
	newStorageLimit := 2048
	checkUpgradeStatusObj := checkUpgradeStatusObject{
		StorageLimit:     newStorageLimit,
		DurationInMonths: models.DefaultMonthsPerSubscription,
		MetadataKeys:     []string{utils.GenerateFileHandle()},
		FileHandles:      []string{utils.GenerateFileHandle()},
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, checkUpgradeStatusObj)

	checkUpgradeStatusReq := checkUpgradeStatusReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)

	account.CreatedAt = time.Now().Add(time.Hour * 24 * (365 / 2) * -1)
	account.PaymentStatus = models.PaymentRetrievalComplete
	models.DB.Save(&account)

	makeCompletedFileForTest(checkUpgradeStatusObj.FileHandles[0], account.ExpirationDate(), v.PublicKey)
	makeMetadataForTest(checkUpgradeStatusObj.MetadataKeys[0], v.PublicKey)

	completedFileStart, err := models.GetCompletedFileByFileID(checkUpgradeStatusObj.FileHandles[0])
	assert.Nil(t, err)

	models.EthWrapper.TransferToken = func(from common.Address, privateKey *ecdsa.PrivateKey, to common.Address,
		opqAmount big.Int, gasPrice *big.Int) (bool, string, int64) {
		return true, "", 1
	}
	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return true, nil
	}

	upgradeCostInUSD, _ := account.UpgradeCostInUSD(checkUpgradeStatusObj.StorageLimit,
		checkUpgradeStatusObj.DurationInMonths)
	stripeToken := services.RandTestStripeToken()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	_, stripePayment, err := createChargeAndStripePayment(c, upgradeCostInUSD, account, stripeToken)
	payUpgradeCostWithStripe(c, stripePayment, account, createStripePaymentObject{
		StorageLimit:     newStorageLimit,
		DurationInMonths: models.DefaultMonthsPerSubscription,
	})
	account.UpdatePaymentViaStripe()

	w := httpPostRequestHelperForTest(t, AccountUpgradePath, checkUpgradeStatusReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	completedFileEnd, err := models.GetCompletedFileByFileID(checkUpgradeStatusObj.FileHandles[0])
	assert.Nil(t, err)

	account, err = models.GetAccountById(account.AccountID)
	assert.Nil(t, err)

	assert.NotEqual(t, completedFileStart.ExpiredAt, completedFileEnd.ExpiredAt)
	assert.Equal(t, completedFileEnd.ExpiredAt, account.ExpirationDate())

	assert.Equal(t, newStorageLimit, int(account.StorageLimit))
	assert.True(t, account.MonthsInSubscription > models.DefaultMonthsPerSubscription)
	assert.Contains(t, w.Body.String(), `Success with Stripe`)
}

func makeCompletedFileForTest(handle string, expirationDate time.Time, key string) {
	completedFile := models.CompletedFile{
		FileID:         handle,
		FileSizeInByte: 150,
		ExpiredAt:      expirationDate,
	}
	modifierHash, _ := utils.HashString(key + completedFile.FileID)
	completedFile.ModifierHash = modifierHash
	models.DB.Save(&completedFile)
}

func makeMetadataForTest(metadataKey string, key string) {
	ttl := utils.TestValueTimeToLive

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	permissionHash, _ := getPermissionHash(key, metadataKey, c)

	permissionHashKey := getPermissionHashKeyForBadger(metadataKey)
	utils.BatchSet(&utils.KVPairs{
		metadataKey:       "",
		permissionHashKey: permissionHash,
	}, ttl)
}
