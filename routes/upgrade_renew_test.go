package routes

import (
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_Renew_And_Upgrade_Accounts(t *testing.T) {
	setupTests(t)
}

func Test_Renew_And_Upgrade_Keeps_Expiration_Year(t *testing.T) {
	models.DeleteAccountsForTest(t)
	models.DeleteRenewalsForTest(t)
	models.DeleteUpgradesForTest(t)

	/*

		First do a renewal and check that the expiration has moved forward 1 year

	*/

	checkRenewalStatusObj := checkRenewalStatusObject{
		MetadataKeys: []string{utils.GenerateFileHandle()},
		FileHandles:  []string{utils.GenerateFileHandle()},
		NetworkID:    utils.TestNetworkID,
	}

	v, b, privateKey := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, checkRenewalStatusObj)

	checkRenewalStatusReq := checkRenewalStatusReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)

	account.CreatedAt = time.Now().Add(time.Hour * 24 * 360 * -1)
	account.PaymentStatus = models.PaymentRetrievalComplete
	models.DB.Save(&account)

	originalMonthsInSubscription := account.MonthsInSubscription

	CreateRenewalForTest(t, account)

	renewals, err := models.GetRenewalsFromAccountID(account.AccountID)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, renewals[0].PaymentStatus)

	makeCompletedFileForTest(checkRenewalStatusObj.FileHandles[0], account.ExpirationDate(), v.PublicKey)
	makeMetadataForTest(checkRenewalStatusObj.MetadataKeys[0], v.PublicKey)

	completedFileStart, err := models.GetCompletedFileByFileID(checkRenewalStatusObj.FileHandles[0])
	assert.Nil(t, err)

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int, networkID uint) (bool, error) {
		return true, nil
	}

	w := httpPostRequestHelperForTest(t, AccountRenewPath, "v1", checkRenewalStatusReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	completedFileEnd, err := models.GetCompletedFileByFileID(checkRenewalStatusObj.FileHandles[0])
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

	/*

		Now do an upgrade and check that the expiration date is at least as far in the future
		as it was when the renewal was done.

	*/

	afterRenewMonthsInSubscription := account.MonthsInSubscription
	afterRenewExpirationDate := account.ExpirationDate()

	newStorageLimit := 1024

	checkUpgradeStatusObj := checkUpgradeStatusObject{
		StorageLimit: newStorageLimit,
		//DurationInMonths: models.DefaultMonthsPerSubscription,
		DurationInMonths: account.MonthsInSubscription,
		MetadataKeys:     checkRenewalStatusObj.MetadataKeys,
		FileHandles:      checkRenewalStatusObj.FileHandles,
		NetworkID:        utils.TestNetworkID,
	}

	v, b = returnValidVerificationAndRequestBody(t, checkUpgradeStatusObj, privateKey)

	checkUpgradeStatusReq := checkUpgradeStatusReq{
		verification: v,
		requestBody:  b,
	}

	CreateUpgradeForTest(t, account, newStorageLimit)

	originalStorageLimit := int(account.StorageLimit)

	upgrade, err := models.GetUpgradeFromAccountIDAndStorageLimits(account.AccountID, newStorageLimit, originalStorageLimit)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentInProgress, upgrade.PaymentStatus)

	completedFileStart, err = models.GetCompletedFileByFileID(checkUpgradeStatusObj.FileHandles[0])
	assert.Nil(t, err)

	w = httpPostRequestHelperForTest(t, AccountUpgradePath, "v1", checkUpgradeStatusReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)

	completedFileEnd, err = models.GetCompletedFileByFileID(checkUpgradeStatusObj.FileHandles[0])
	assert.Nil(t, err)

	account, err = models.GetAccountById(account.AccountID)
	assert.Nil(t, err)

	assert.True(t, completedFileEnd.ExpiredAt.Unix() >= completedFileStart.ExpiredAt.Unix())
	assert.Equal(t, completedFileEnd.ExpiredAt, account.ExpirationDate())

	assert.Equal(t, newStorageLimit, int(account.StorageLimit))
	assert.True(t, account.MonthsInSubscription >= afterRenewMonthsInSubscription)
	assert.True(t, account.ExpirationDate().Unix() >= afterRenewExpirationDate.Unix())
	assert.Contains(t, w.Body.String(), `Success with OPCT`)

	upgrade, err = models.GetUpgradeFromAccountIDAndStorageLimits(account.AccountID, newStorageLimit, originalStorageLimit)
	assert.Nil(t, err)
	assert.Equal(t, models.InitialPaymentReceived, upgrade.PaymentStatus)
}
