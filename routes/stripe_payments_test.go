package routes

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
	"math/big"
	"net/http"
	"testing"
	"time"
)

func Test_Init_Stripe_Payments(t *testing.T) {
	setupTests(t)
	err := services.InitStripe()
	assert.Nil(t, err)
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
		return true, nil
	}
	EthWrapper.TransferToken = func(from common.Address, privateKey *ecdsa.PrivateKey, to common.Address,
		opqAmount big.Int, gasPrice *big.Int) (bool, string, int64) {
		return true, "", 1
	}

	w := httpPostRequestHelperForTest(t, StripeCreatePath, post)

	assert.Equal(t, http.StatusOK, w.Code)
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

	w := httpPostRequestHelperForTest(t, StripeCreatePath, post)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "account is already paid for")
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
		return true, nil
	}
	EthWrapper.TransferToken = func(from common.Address, privateKey *ecdsa.PrivateKey, to common.Address,
		opqAmount big.Int, gasPrice *big.Int) (bool, string, int64) {
		return false, "", 1
	}

	w := httpPostRequestHelperForTest(t, StripeCreatePath, post)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
