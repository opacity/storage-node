package routes

import (
	"crypto/ecdsa"
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

func Test_Init_Stripe_Payments(t *testing.T) {
	setupTests(t)
}

func Test_Successful_Stripe_Payment(t *testing.T) {
	models.DeleteStripePaymentsForTest(t)
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	accountID, _ := utils.HashString(utils.PubkeyCompressedToHex(privateKey.PublicKey))
	CreateUnpaidAccountForTest(t, accountID)

	stripeTokenBody := createStripePaymentObject{
		StripeToken: services.RandTestStripeToken(),
		Timestamp:   time.Now().Unix(),
	}
	v, b := returnValidVerificationAndRequestBody(t, stripeTokenBody, privateKey)

	post := createStripePaymentReq{
		verification: v,
		requestBody:  b,
	}

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}
	models.EthWrapper.TransferToken = func(from common.Address, privateKey *ecdsa.PrivateKey, to common.Address,
		opqAmount big.Int, gasPrice *big.Int) (bool, string, int64) {
		return true, "", 1
	}

	w := httpPostRequestHelperForTest(t, StripeCreatePath, post)

	assert.Equal(t, http.StatusOK, w.Code)

	account, _ := models.GetAccountById(accountID)
	assert.Equal(t, models.PaymentMethodWithCreditCard, account.PaymentMethod)
}

func Test_Fails_If_Account_Does_Not_Exist(t *testing.T) {
	models.DeleteStripePaymentsForTest(t)
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)

	stripeTokenBody := createStripePaymentObject{
		StripeToken: services.RandTestStripeToken(),
		Timestamp:   time.Now().Unix(),
	}
	v, b := returnValidVerificationAndRequestBody(t, stripeTokenBody, privateKey)

	post := createStripePaymentReq{
		verification: v,
		requestBody:  b,
	}

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	w := httpPostRequestHelperForTest(t, StripeCreatePath, post)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), noAccountWithThatID)
}

func Test_Fails_If_Account_Is_Paid(t *testing.T) {
	models.DeleteStripePaymentsForTest(t)
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	accountID, _ := utils.HashString(utils.PubkeyCompressedToHex(privateKey.PublicKey))
	CreatePaidAccountForTest(t, accountID)

	stripeTokenBody := createStripePaymentObject{
		StripeToken: services.RandTestStripeToken(),
		Timestamp:   time.Now().Unix(),
	}
	v, b := returnValidVerificationAndRequestBody(t, stripeTokenBody, privateKey)

	post := createStripePaymentReq{
		verification: v,
		requestBody:  b,
	}

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return true, nil
	}

	w := httpPostRequestHelperForTest(t, StripeCreatePath, post)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "account is already paid for")
}

func Test_Fails_If_Account_Is_Free(t *testing.T) {
	models.DeleteStripePaymentsForTest(t)
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	accountID, _ := utils.HashString(utils.PubkeyCompressedToHex(privateKey.PublicKey))
	account := CreateUnpaidAccountForTest(t, accountID)
	account.StorageLimit = models.StorageLimitType(utils.Env.Plans[10].StorageInGB)
	err = models.DB.Save(&account).Error
	assert.Nil(t, err)

	stripeTokenBody := createStripePaymentObject{
		StripeToken: services.RandTestStripeToken(),
		Timestamp:   time.Now().Unix(),
	}
	v, b := returnValidVerificationAndRequestBody(t, stripeTokenBody, privateKey)

	post := createStripePaymentReq{
		verification: v,
		requestBody:  b,
	}

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	w := httpPostRequestHelperForTest(t, StripeCreatePath, post)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "cannot create stripe charge for less than $0.50")
}

func Test_Unsuccessful_Token_Transfer_Returns_Error(t *testing.T) {
	models.DeleteStripePaymentsForTest(t)
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	accountID, _ := utils.HashString(utils.PubkeyCompressedToHex(privateKey.PublicKey))
	CreateUnpaidAccountForTest(t, accountID)

	stripeTokenBody := createStripePaymentObject{
		StripeToken: services.RandTestStripeToken(),
		Timestamp:   time.Now().Unix(),
	}
	v, b := returnValidVerificationAndRequestBody(t, stripeTokenBody, privateKey)

	post := createStripePaymentReq{
		verification: v,
		requestBody:  b,
	}

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}
	models.EthWrapper.TransferToken = func(from common.Address, privateKey *ecdsa.PrivateKey, to common.Address,
		opqAmount big.Int, gasPrice *big.Int) (bool, string, int64) {
		return false, "", 1
	}

	w := httpPostRequestHelperForTest(t, StripeCreatePath, post)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func Test_Successful_Stripe_Payment_For_Upgrade(t *testing.T) {
	models.DeleteStripePaymentsForTest(t)
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	accountID, _ := utils.HashString(utils.PubkeyCompressedToHex(privateKey.PublicKey))
	account := CreatePaidAccountForTest(t, accountID)
	account.CreatedAt = time.Now().Add(time.Hour * 24 * (365 / 2) * -1)
	models.DB.Save(&account)

	newStorageLimit := 1024

	upgrade := CreateUpgradeForTest(t, account, newStorageLimit)
	models.DB.Save(&upgrade)

	oldExpirationDate := account.ExpirationDate()
	upgradeCostInUSD, err := account.UpgradeCostInUSD(newStorageLimit, models.DefaultMonthsPerSubscription)
	assert.Nil(t, err)

	stripeTokenBody := createStripePaymentObject{
		StripeToken:      services.RandTestStripeToken(),
		Timestamp:        time.Now().Unix(),
		UpgradeAccount:   true,
		StorageLimit:     newStorageLimit,
		DurationInMonths: models.DefaultMonthsPerSubscription,
	}
	v, b := returnValidVerificationAndRequestBody(t, stripeTokenBody, privateKey)

	post := createStripePaymentReq{
		verification: v,
		requestBody:  b,
	}

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}
	models.EthWrapper.TransferToken = func(from common.Address, privateKey *ecdsa.PrivateKey, to common.Address,
		opqAmount big.Int, gasPrice *big.Int) (bool, string, int64) {
		return true, "", 1
	}

	w := httpPostRequestHelperForTest(t, StripeCreatePath, post)

	assert.Equal(t, http.StatusOK, w.Code)

	account, _ = models.GetAccountById(accountID)
	assert.Equal(t, models.PaymentMethodWithCreditCard, account.PaymentMethod)
	y1, m1, d1 := oldExpirationDate.Date()
	y2, m2, d2 := account.ExpirationDate().Date()
	assert.Equal(t, y1, y2)
	assert.Equal(t, m1, m2)
	assert.Equal(t, d1, d2)
	assert.Equal(t, models.InitialPaymentInProgress, account.PaymentStatus)

	stripePayment, err := models.GetStripePaymentByAccountId(account.AccountID)
	assert.Nil(t, err)

	amount, err := services.CheckChargeAmount(stripePayment.ChargeID)
	assert.Nil(t, err)

	assert.Equal(t, upgradeCostInUSD, amount)
}
