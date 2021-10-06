package routes

import (
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

func Test_Init_Upgrade_Accounts(t *testing.T) {
	setupTests(t)
}

func Test_GetAccountUpgradeInvoiceHandler_Returns_Invoice(t *testing.T) {
	models.DeleteAccountsForTest(t)
	models.DeleteUpgradesForTest(t)

	getInvoiceObj := getUpgradeAccountInvoiceObject{
		PlanID: 4,
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getInvoiceObj)

	getInvoiceReq := getUpgradeAccountInvoiceReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)

	professionalPlan, err := models.GetPlanInfoByID(3)
	assert.Nil(t, err)
	account.PlanInfo = professionalPlan
	account.PlanInfoID = professionalPlan.ID

	account.CreatedAt = time.Now().Add(time.Hour * 24 * (365 / 2) * -1)
	models.DB.Save(&account)

	w := httpPostRequestHelperForTest(t, AccountUpgradeInvoicePath, "v1", getInvoiceReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	//assert.Contains(t, w.Body.String(), `"usdInvoice":100`)
	assert.Contains(t, w.Body.String(), `"opctInvoice":{"cost":24,`)
}

func Test_CheckUpgradeStatusHandler_Returns_Status_OPCT_Upgrade_Success(t *testing.T) {
	models.DeleteAccountsForTest(t)
	models.DeleteUpgradesForTest(t)

	newPlanID := uint(3)

	checkUpgradeStatusObj := checkUpgradeStatusObject{
		PlanID:       newPlanID,
		MetadataKeys: []string{utils.GenerateFileHandle()},
		FileHandles:  []string{utils.GenerateFileHandle()},
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

	CreateUpgradeForTest(t, account, newPlanID)

	originalPlanID := account.PlanInfo.ID

	upgrade, err := models.GetUpgradeFromAccountIDAndPlans(account.AccountID, newPlanID, originalPlanID)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)

	makeCompletedFileForTest(checkUpgradeStatusObj.FileHandles[0], account.ExpirationDate(), v.PublicKey)
	makeMetadataForTest(checkUpgradeStatusObj.MetadataKeys[0], v.PublicKey)

	completedFileStart, err := models.GetCompletedFileByFileID(checkUpgradeStatusObj.FileHandles[0])
	assert.Nil(t, err)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return true, utils.TestNetworkID, nil
	}

	w := httpPostRequestHelperForTest(t, AccountUpgradePath, "v1", checkUpgradeStatusReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	completedFileEnd, err := models.GetCompletedFileByFileID(checkUpgradeStatusObj.FileHandles[0])
	assert.Nil(t, err)

	account, err = models.GetAccountById(account.AccountID)
	assert.Nil(t, err)

	assert.NotEqual(t, completedFileStart.ExpiredAt, completedFileEnd.ExpiredAt)
	assert.Equal(t, completedFileEnd.ExpiredAt, account.ExpirationDate())

	assert.Equal(t, newPlanID, account.PlanInfo.ID)
	assert.True(t, account.MonthsInSubscription > models.DefaultMonthsPerSubscription)
	assert.Contains(t, w.Body.String(), `Success with OPCT`)

	upgrade, err = models.GetUpgradeFromAccountIDAndPlans(account.AccountID, newPlanID, originalPlanID)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentReceived, upgrade.PaymentStatus)
}

func Test_CheckUpgradeStatusHandler_Returns_Status_OPCT_Upgrade_Still_Pending(t *testing.T) {
	setupTests(t)

	models.DeleteAccountsForTest(t)
	models.DeleteUpgradesForTest(t)
	models.DeleteStripePaymentsForTest(t)

	newPlanID := uint(3)

	checkUpgradeStatusObj := checkUpgradeStatusObject{
		PlanID:       newPlanID,
		MetadataKeys: []string{utils.GenerateFileHandle()},
		FileHandles:  []string{utils.GenerateFileHandle()},
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

	CreateUpgradeForTest(t, account, newPlanID)

	originalPlanID := account.PlanInfo.ID

	upgrade, err := models.GetUpgradeFromAccountIDAndPlans(account.AccountID, newPlanID, originalPlanID)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return false, utils.TestNetworkID, nil
	}

	w := httpPostRequestHelperForTest(t, AccountUpgradePath, "v1", checkUpgradeStatusReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	account, err = models.GetAccountById(account.AccountID)
	assert.Nil(t, err)
	assert.NotEqual(t, newPlanID, account.PlanInfo.ID)
	assert.NotEqual(t, models.InitialPaymentReceived, account.PaymentStatus)
	assert.False(t, account.MonthsInSubscription > models.DefaultMonthsPerSubscription)
	assert.Contains(t, w.Body.String(), `Incomplete`)

	upgrade, err = models.GetUpgradeFromAccountIDAndPlans(account.AccountID, newPlanID, originalPlanID)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)
}

// TODO: uncomment out if/when we support stripe for upgrade
//func Test_CheckUpgradeStatusHandler_Returns_Status_Stripe_Upgrade(t *testing.T) {
//	models.DeleteAccountsForTest(t)
//	models.DeleteUpgradesForTest(t)
//
//	newStorageLimit := 1024
//
//	checkUpgradeStatusObj := checkUpgradeStatusObject{
//		StorageLimit:     newStorageLimit,
//		DurationInMonths: models.DefaultMonthsPerSubscription,
//		MetadataKeys:     []string{utils.GenerateFileHandle()},
//		FileHandles:      []string{utils.GenerateFileHandle()},
//	}
//
//	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, checkUpgradeStatusObj)
//
//	checkUpgradeStatusReq := checkUpgradeStatusReq{
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
//	CreateUpgradeForTest(t, account, newStorageLimit)
//
//	originalStorageLimit := int(account.StorageLimit)
//
//	upgrade, err := models.GetUpgradeFromAccountIDAndStorageLimits(account.AccountID, newStorageLimit, originalStorageLimit)
//	assert.Nil(t, err)
//	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)
//
//	makeCompletedFileForTest(checkUpgradeStatusObj.FileHandles[0], account.ExpirationDate(), v.PublicKey)
//	makeMetadataForTest(checkUpgradeStatusObj.MetadataKeys[0], v.PublicKey)
//
//	completedFileStart, err := models.GetCompletedFileByFileID(checkUpgradeStatusObj.FileHandles[0])
//	assert.Nil(t, err)
//
//	services.EthOpsWrapper.TransferToken = func(ethWrapper *services.Eth, from common.Address, privateKey *ecdsa.PrivateKey, to common.Address,
//		opctAmount big.Int, gasPrice *big.Int) (bool, string, int64) {
//		return true, "", 1
//	}
//	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
//		return false, nil
//	}
//
//	upgradeCostInUSD, _ := account.UpgradeCostInUSD(checkUpgradeStatusObj.StorageLimit,
//		checkUpgradeStatusObj.DurationInMonths)
//	stripeToken := services.RandTestStripeToken()
//	c, _ := gin.CreateTestContext(httptest.NewRecorder())
//	_, stripePayment, err := createChargeAndStripePayment(c, upgradeCostInUSD, account, createStripePaymentObject{
//		StripeToken:    stripeToken,
//		UpgradeAccount: true,
//	})
//	err = payUpgradeCostWithStripe(c, stripePayment, account, createStripePaymentObject{
//		StorageLimit:     newStorageLimit,
//		DurationInMonths: models.DefaultMonthsPerSubscription,
//	})
//	assert.Nil(t, err)
//	account.UpdatePaymentViaStripe()
//
//	w := httpPostRequestHelperForTest(t, AccountUpgradePath, checkUpgradeStatusReq)
//	// Check to see if the response was what you expected
//	assert.Equal(t, http.StatusOK, w.Code)
//
//	completedFileEnd, err := models.GetCompletedFileByFileID(checkUpgradeStatusObj.FileHandles[0])
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
//	upgrade, err = models.GetUpgradeFromAccountIDAndStorageLimits(account.AccountID, newStorageLimit, originalStorageLimit)
//	assert.Nil(t, err)
//	// This is in progress because we returned false for the CheckIfPaid mock
//	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)
//}

func Test_CheckUpgradeStatusHandler_Multiple_Upgrades(t *testing.T) {
	models.DeleteAccountsForTest(t)
	models.DeleteUpgradesForTest(t)

	newPlanID := uint(3)
	newPlanID2 := uint(4)

	checkUpgradeStatusObj := checkUpgradeStatusObject{
		PlanID:       newPlanID2,
		MetadataKeys: []string{utils.GenerateFileHandle()},
		FileHandles:  []string{utils.GenerateFileHandle()},
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

	CreateUpgradeForTest(t, account, newPlanID)

	originalPlanID := account.PlanInfo.ID

	upgrade, err := models.GetUpgradeFromAccountIDAndPlans(account.AccountID, newPlanID, originalPlanID)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)

	upgrade2 := returnUpgradeForTest(t, account, newPlanID2)

	newPlanInfo2, _ := models.GetPlanInfoByID(newPlanID2)
	upgradeCostInOPCT, _ := account.UpgradeCostInOPCT(newPlanInfo2)

	upgrade2.OpctCost = upgradeCostInOPCT

	err = models.DB.Create(&upgrade2).Error
	assert.Nil(t, err)

	makeCompletedFileForTest(checkUpgradeStatusObj.FileHandles[0], account.ExpirationDate(), v.PublicKey)
	makeMetadataForTest(checkUpgradeStatusObj.MetadataKeys[0], v.PublicKey)

	completedFileStart, err := models.GetCompletedFileByFileID(checkUpgradeStatusObj.FileHandles[0])
	assert.Nil(t, err)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return true, utils.TestNetworkID, nil
	}

	w := httpPostRequestHelperForTest(t, AccountUpgradePath, "v1", checkUpgradeStatusReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	completedFileEnd, err := models.GetCompletedFileByFileID(checkUpgradeStatusObj.FileHandles[0])
	assert.Nil(t, err)

	account, err = models.GetAccountById(account.AccountID)
	assert.Nil(t, err)

	assert.NotEqual(t, completedFileStart.ExpiredAt, completedFileEnd.ExpiredAt)
	assert.Equal(t, completedFileEnd.ExpiredAt, account.ExpirationDate())

	assert.Equal(t, newPlanID2, account.PlanInfo.ID)
	assert.True(t, account.MonthsInSubscription > models.DefaultMonthsPerSubscription)
	assert.Contains(t, w.Body.String(), `Success with OPCT`)

	upgrade, err = models.GetUpgradeFromAccountIDAndPlans(account.AccountID, newPlanID, originalPlanID)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)

	upgrade, err = models.GetUpgradeFromAccountIDAndPlans(account.AccountID, newPlanID2, originalPlanID)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentReceived, upgrade.PaymentStatus)
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

func CreateUpgradeForTest(t *testing.T, account models.Account, newPlanId uint) models.Upgrade {
	upgrade := returnUpgradeForTest(t, account, newPlanId)

	if err := models.DB.Create(&upgrade).Error; err != nil {
		t.Fatalf("should have created upgrade but didn't: " + err.Error())
	}

	return upgrade
}

func returnUpgradeForTest(t *testing.T, account models.Account, newPlanId uint) models.Upgrade {
	abortIfNotTesting(t)

	ethAddress, privateKey := services.GenerateWallet()

	plan, _ := models.GetPlanInfoByID(newPlanId)
	upgradeCostInOPCT, _ := account.UpgradeCostInOPCT(plan)

	return models.Upgrade{
		AccountID:     account.AccountID,
		NewPlanInfoID: plan.ID,
		NewPlanInfo:   plan,
		OldPlanInfoID: account.PlanInfo.ID,
		PaymentStatus: models.InitialPaymentInProgress,
		OpctCost:      upgradeCostInOPCT,
		//UsdCost:          upgradeCostInUSD,
		EthAddress:    ethAddress.String(),
		EthPrivateKey: hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, account.AccountID)),
		NetworkIdPaid: utils.TestNetworkID,
	}
}
