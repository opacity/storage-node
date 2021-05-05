package routes

import (
	"math/big"
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

func Test_verifyIfPaidWithContext_account_status_already_paid(t *testing.T) {
	models.DeleteAccountsForTest()
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	accountID, _ := utils.HashString(utils.PubkeyCompressedToHex(privateKey.PublicKey))
	account := CreatePaidAccountForTest(t, accountID)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	err = verifyIfPaidWithContext(account, c)

	assert.Nil(t, err)
}

func Test_verifyIfPaidWithContext_account_opct_balance_has_arrived(t *testing.T) {
	models.DeleteAccountsForTest()
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	accountID, _ := utils.HashString(utils.PubkeyCompressedToHex(privateKey.PublicKey))
	account := CreateUnpaidAccountForTest(t, accountID)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return true, nil
	}

	err = verifyIfPaidWithContext(account, c)

	assert.Nil(t, err)
}

func Test_verifyIfPaidWithContext_stripe_payment_has_been_paid(t *testing.T) {
	models.DeleteAccountsForTest()
	models.DeleteStripePaymentsForTest()
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	accountID, _ := utils.HashString(utils.PubkeyCompressedToHex(privateKey.PublicKey))
	account := CreateUnpaidAccountForTest(t, accountID)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	stripeToken := services.RandTestStripeToken()
	charge, _ := services.CreateCharge(float64(utils.Env.Plans[int(account.StorageLimit)].CostInUSD), stripeToken, account.AccountID)

	stripePayment := models.StripePayment{
		StripeToken: stripeToken,
		AccountID:   account.AccountID,
		ChargeID:    charge.ID,
	}

	models.DB.Create(&stripePayment)

	err = verifyIfPaidWithContext(account, c)

	assert.Nil(t, err)
}

func Test_verifyIfPaidWithContext_account_not_paid_and_no_stripe_payment(t *testing.T) {
	models.DeleteAccountsForTest()
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	accountID, _ := utils.HashString(utils.PubkeyCompressedToHex(privateKey.PublicKey))
	account := CreateUnpaidAccountForTest(t, accountID)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	err = verifyIfPaidWithContext(account, c)

	assert.NotNil(t, err)
}

func Test_verifyValidStorageLimit(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	err := verifyValidStorageLimit(128, c)
	assert.Nil(t, err)

	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	err = verifyValidStorageLimit(129, c)
	assert.NotNil(t, err)
}

func Test_verifyAccountStillActive(t *testing.T) {
	models.DeleteAccountsForTest()
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	accountID, _ := utils.HashString(utils.PubkeyCompressedToHex(privateKey.PublicKey))
	account := CreatePaidAccountForTest(t, accountID)

	stillActive := verifyAccountStillActive(account)
	assert.True(t, stillActive)

	account.CreatedAt = time.Now().Add(time.Hour * 24 * 366 * -1)
	err = models.DB.Save(&account).Error
	assert.Nil(t, err)

	stillActive = verifyAccountStillActive(account)
	assert.False(t, stillActive)

	account.StorageLimit = 10
	err = models.DB.Save(&account).Error
	assert.Nil(t, err)

	stillActive = verifyAccountStillActive(account)
	assert.True(t, stillActive)
}

func Test_verifyRenewEligible(t *testing.T) {
	models.DeleteAccountsForTest()
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	accountID, _ := utils.HashString(utils.PubkeyCompressedToHex(privateKey.PublicKey))
	account := CreatePaidAccountForTest(t, accountID)
	account.MonthsInSubscription = 5
	err = models.DB.Save(&account).Error
	assert.Nil(t, err)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	err = verifyRenewEligible(account, c)
	assert.Nil(t, err)

	account.MonthsInSubscription = 24
	err = models.DB.Save(&account).Error
	assert.Nil(t, err)

	err = verifyRenewEligible(account, c)
	assert.Error(t, err)
}
