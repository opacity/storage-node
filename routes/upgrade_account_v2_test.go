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

	businessPlan, err := models.GetPlanInfoByID(4)
	assert.Nil(t, err)

	getInvoiceObj := getUpgradeV2AccountInvoiceObject{
		PlanID: businessPlan.ID,
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getInvoiceObj)

	getInvoiceReq := getUpgradeV2AccountInvoiceReq{
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

	w := httpPostRequestHelperForTest(t, AccountUpgradeV2InvoicePath, "v2", getInvoiceReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	//assert.Contains(t, w.Body.String(), `"usdInvoice":100`)
	assert.Contains(t, w.Body.String(), `"opctInvoice":{"cost":24,`)
}

func Test_CheckUpgradeV2StatusHandler_Returns_Status_OPCT_UpgradeV2_Success(t *testing.T) {
	t.SkipNow()

	models.DeleteAccountsForTest(t)
	models.DeleteUpgradesForTest(t)

	newPlanID := uint(3)

	checkUpgradeV2StatusObj := checkUpgradeV2StatusObject{
		PlanID:       newPlanID,
		MetadataKeys: []string{utils.GenerateMetadataV2Key()},
		FileHandles:  []string{utils.GenerateMetadataV2Key()},
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

	CreateUpgradeV2ForTest(t, account, newPlanID)

	originalPlanID := account.PlanInfo.ID

	upgrade, err := models.GetUpgradeFromAccountIDAndPlans(account.AccountID, newPlanID, originalPlanID)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)

	makeCompletedFileForTest(checkUpgradeV2StatusObj.FileHandles[0], account.ExpirationDate(), v.PublicKey)
	makeMetadataForTest(checkUpgradeV2StatusObj.MetadataKeys[0], v.PublicKey)

	completedFileStart, err := models.GetCompletedFileByFileID(checkUpgradeV2StatusObj.FileHandles[0])
	assert.Nil(t, err)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return true, utils.TestNetworkID, nil
	}

	w := httpPostRequestHelperForTest(t, AccountUpgradeV2Path, "v2", checkUpgradeV2StatusReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	completedFileEnd, err := models.GetCompletedFileByFileID(checkUpgradeV2StatusObj.FileHandles[0])
	assert.Nil(t, err)

	account, err = models.GetAccountById(account.AccountID)
	assert.Nil(t, err)

	assert.NotEqual(t, completedFileStart.ExpiredAt, completedFileEnd.ExpiredAt)
	assert.Equal(t, completedFileEnd.ExpiredAt, account.ExpirationDate())

	assert.Equal(t, newPlanID, account.PlanInfo.ID)
	assert.True(t, account.MonthsInSubscription > int(account.PlanInfo.MonthsInSubscription))
	assert.Contains(t, w.Body.String(), `Success with OPCT`)

	upgrade, err = models.GetUpgradeFromAccountIDAndPlans(account.AccountID, newPlanID, originalPlanID)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentReceived, upgrade.PaymentStatus)
}

func Test_CheckUpgradeV2StatusHandler_Returns_Status_OPCT_UpgradeV2_Still_Pending(t *testing.T) {
	models.DeleteAccountsForTest(t)
	models.DeleteUpgradesForTest(t)
	models.DeleteStripePaymentsForTest(t)

	newPlanID := uint(3)

	checkUpgradeV2StatusObj := checkUpgradeV2StatusObject{
		PlanID:       newPlanID,
		MetadataKeys: []string{utils.GenerateMetadataV2Key()},
		FileHandles:  []string{utils.GenerateMetadataV2Key()},
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

	CreateUpgradeV2ForTest(t, account, newPlanID)

	originalPlanID := account.PlanInfo.ID

	upgrade, err := models.GetUpgradeFromAccountIDAndPlans(account.AccountID, newPlanID, originalPlanID)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return false, utils.TestNetworkID, nil
	}

	w := httpPostRequestHelperForTest(t, AccountUpgradeV2Path, "v2", checkUpgradeV2StatusReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	account, err = models.GetAccountById(account.AccountID)
	assert.Nil(t, err)
	assert.NotEqual(t, newPlanID, account.PlanInfo.ID)
	assert.NotEqual(t, models.InitialPaymentReceived, account.PaymentStatus)
	assert.False(t, account.MonthsInSubscription > int(account.PlanInfo.MonthsInSubscription))
	assert.Contains(t, w.Body.String(), `Incomplete`)

	upgrade, err = models.GetUpgradeFromAccountIDAndPlans(account.AccountID, newPlanID, originalPlanID)
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
//	services.EthOpsWrapper.TransferToken = func(ethWrapper *services.Eth, from common.Address, privateKey *ecdsa.PrivateKey, to common.Address,
//		opctAmount big.Int, gasPrice *big.Int) (bool, string, int64) {
//		return true, "", 1
//	}
//	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
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
	t.SkipNow()

	models.DeleteAccountsForTest(t)
	models.DeleteUpgradesForTest(t)

	newPlanID := uint(3)
	newPlanID2 := uint(4)

	checkUpgradeV2StatusObj := checkUpgradeV2StatusObject{
		PlanID:       newPlanID2,
		MetadataKeys: []string{utils.GenerateMetadataV2Key()},
		FileHandles:  []string{utils.GenerateMetadataV2Key()},
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

	CreateUpgradeV2ForTest(t, account, newPlanID)

	originalPlanID := account.PlanInfo.ID

	upgrade, err := models.GetUpgradeFromAccountIDAndPlans(account.AccountID, newPlanID, originalPlanID)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)

	upgrade2 := returnUpgradeV2ForTest(t, account, newPlanID2)

	newPlanInfo2, _ := models.GetPlanInfoByID(newPlanID2)
	upgradeCostInOPCT, _ := account.UpgradeCostInOPCT(newPlanInfo2)

	upgrade2.OpctCost = upgradeCostInOPCT

	err = models.DB.Create(&upgrade2).Error
	assert.Nil(t, err)

	makeCompletedFileForTest(checkUpgradeV2StatusObj.FileHandles[0], account.ExpirationDate(), v.PublicKey)
	makeMetadataForTest(checkUpgradeV2StatusObj.MetadataKeys[0], v.PublicKey)

	completedFileStart, err := models.GetCompletedFileByFileID(checkUpgradeV2StatusObj.FileHandles[0])
	assert.Nil(t, err)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return true, utils.TestNetworkID, nil
	}

	w := httpPostRequestHelperForTest(t, AccountUpgradeV2Path, "v2", checkUpgradeV2StatusReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	completedFileEnd, err := models.GetCompletedFileByFileID(checkUpgradeV2StatusObj.FileHandles[0])
	assert.Nil(t, err)

	account, err = models.GetAccountById(account.AccountID)
	assert.Nil(t, err)

	assert.NotEqual(t, completedFileStart.ExpiredAt, completedFileEnd.ExpiredAt)
	assert.Equal(t, completedFileEnd.ExpiredAt, account.ExpirationDate())

	assert.Equal(t, newPlanID2, account.PlanInfo.ID)
	assert.True(t, account.MonthsInSubscription > int(account.PlanInfo.MonthsInSubscription))
	assert.Contains(t, w.Body.String(), `Success with OPCT`)

	upgrade, err = models.GetUpgradeFromAccountIDAndPlans(account.AccountID, newPlanID, originalPlanID)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)

	upgrade, err = models.GetUpgradeFromAccountIDAndPlans(account.AccountID, newPlanID2, originalPlanID)
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

func CreateUpgradeV2ForTest(t *testing.T, account models.Account, newPlanId uint) models.Upgrade {
	upgrade := returnUpgradeV2ForTest(t, account, newPlanId)

	if err := models.DB.Create(&upgrade).Error; err != nil {
		t.Fatalf("should have created upgrade but didn't: " + err.Error())
	}

	return upgrade
}

func returnUpgradeV2ForTest(t *testing.T, account models.Account, newPlanId uint) models.Upgrade {
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
