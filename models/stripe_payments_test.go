package models

import (
	"testing"

	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
	"math/big"
	"time"
)

func returnValidStripePaymentForTest() StripePayment {
	account := returnValidAccount()
	account.MetadataKey = utils.RandSeqFromRunes(AccountIDLength, []rune("abcdef01234567890"))

	// Add account to DB
	DB.Create(&account)

	return StripePayment{
		StripeToken: services.RandTestStripeToken(),
		AccountID:   account.AccountID,
	}
}

func Test_Init_Stripe_Payments(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.TestDatabaseURL)
	err := services.InitStripe()
	assert.Nil(t, err)
}

func Test_Valid_Stripe_Payment_Passes(t *testing.T) {
	DeleteStripePaymentsForTest(t)
	stripePayment := returnValidStripePaymentForTest()

	if err := DB.Create(&stripePayment).Error; err != nil {
		t.Fatalf("should have created row but didn't: " + err.Error())
	}
}

func Test_Valid_Stripe_Fails_If_No_Account_Exists(t *testing.T) {
	DeleteStripePaymentsForTest(t)
	stripePayment := returnValidStripePaymentForTest()
	account, _ := GetAccountById(stripePayment.AccountID)
	DB.Delete(&account)

	if err := DB.Create(&stripePayment).Error; err == nil {
		t.Fatalf("row creation should have failed")
	}
}

func Test_GetStripePaymentByAccountId(t *testing.T) {
	DeleteStripePaymentsForTest(t)
	stripePayment := returnValidStripePaymentForTest()

	if err := DB.Create(&stripePayment).Error; err != nil {
		t.Fatalf("should have created row but didn't: " + err.Error())
	}

	stripeRowFromDB, err := GetStripePaymentByAccountId(stripePayment.AccountID)
	assert.Nil(t, err)
	assert.Equal(t, stripeRowFromDB.AccountID, stripePayment.AccountID)
	assert.NotEqual(t, "", stripeRowFromDB.AccountID)

	DB.Delete(&stripePayment)

	stripePayment = returnValidStripePaymentForTest()

	stripeRowFromDB, err = GetStripePaymentByAccountId(stripePayment.AccountID)
	assert.NotNil(t, err)
	assert.NotEqual(t, stripeRowFromDB.AccountID, stripePayment.AccountID)
	assert.Equal(t, "", stripeRowFromDB.AccountID)
}

func Test_CheckForPaidStripePayment(t *testing.T) {
	DeleteStripePaymentsForTest(t)
	stripePayment := returnValidStripePaymentForTest()

	if err := DB.Create(&stripePayment).Error; err != nil {
		t.Fatalf("should have created row but didn't: " + err.Error())
	}

	paidWithCreditCard, err := CheckForPaidStripePayment(stripePayment.AccountID)
	assert.False(t, paidWithCreditCard)
	assert.NotNil(t, err)

	charge, _ := services.CreateCharge(10, stripePayment.StripeToken)
	stripePayment.ChargeID = charge.ID
	DB.Save(&stripePayment)

	paidWithCreditCard, err = CheckForPaidStripePayment(stripePayment.AccountID)
	assert.True(t, paidWithCreditCard)
	assert.Nil(t, err)
}

func Test_SendAccountOPQ(t *testing.T) {
	DeleteStripePaymentsForTest(t)
	stripePayment := returnValidStripePaymentForTest()

	if err := DB.Create(&stripePayment).Error; err != nil {
		t.Fatalf("should have created row but didn't: " + err.Error())
	}

	EthWrapper.TransferToken = func(from common.Address, privateKey *ecdsa.PrivateKey, to common.Address,
		opqAmount big.Int, gasPrice *big.Int) (bool, string, int64) {
		return true, "", 1
	}

	assert.Equal(t, OpqTxNotStarted, stripePayment.OpqTxStatus)
	err := stripePayment.SendAccountOPQ()
	assert.Nil(t, err)
	assert.Equal(t, OpqTxInProgress, stripePayment.OpqTxStatus)
}

func Test_CheckOPQTransaction_transaction_complete(t *testing.T) {
	DeleteStripePaymentsForTest(t)
	stripePayment := returnValidStripePaymentForTest()
	stripePayment.OpqTxStatus = OpqTxInProgress

	if err := DB.Create(&stripePayment).Error; err != nil {
		t.Fatalf("should have created row but didn't: " + err.Error())
	}

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return true, nil
	}

	assert.Equal(t, OpqTxInProgress, stripePayment.OpqTxStatus)
	txSuccess, err := stripePayment.CheckOPQTransaction()
	assert.True(t, txSuccess)
	stripeRow, err := GetStripePaymentByAccountId(stripePayment.AccountID)
	assert.NotNil(t, err)
	assert.Equal(t, "", stripeRow.StripeToken)

}

func Test_CheckOPQTransaction_transaction_incomplete(t *testing.T) {
	DeleteStripePaymentsForTest(t)
	stripePayment := returnValidStripePaymentForTest()
	stripePayment.OpqTxStatus = OpqTxInProgress

	if err := DB.Create(&stripePayment).Error; err != nil {
		t.Fatalf("should have created row but didn't: " + err.Error())
	}

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	assert.Equal(t, OpqTxInProgress, stripePayment.OpqTxStatus)
	success, err := stripePayment.CheckOPQTransaction()
	assert.Nil(t, err)
	assert.False(t, success)
	assert.Equal(t, OpqTxInProgress, stripePayment.OpqTxStatus)
}

func Test_RetryIfTimedOut_Not_Timed_Out(t *testing.T) {
	DeleteStripePaymentsForTest(t)
	stripePayment := returnValidStripePaymentForTest()
	stripePayment.OpqTxStatus = OpqTxInProgress

	if err := DB.Create(&stripePayment).Error; err != nil {
		t.Fatalf("should have created row but didn't: " + err.Error())
	}

	retryOccurred := false

	EthWrapper.TransferToken = func(from common.Address, privateKey *ecdsa.PrivateKey, to common.Address,
		opqAmount big.Int, gasPrice *big.Int) (bool, string, int64) {
		retryOccurred = true
		return true, "", 1
	}

	stripePayment.RetryIfTimedOut()

	assert.False(t, retryOccurred)
}

func Test_RetryIfTimedOut_Timed_Out(t *testing.T) {
	DeleteStripePaymentsForTest(t)
	stripePayment := returnValidStripePaymentForTest()
	stripePayment.OpqTxStatus = OpqTxInProgress

	if err := DB.Create(&stripePayment).Error; err != nil {
		t.Fatalf("should have created row but didn't: " + err.Error())
	}

	DB.Model(&stripePayment).UpdateColumn("updated_at", time.Now().Add(-1*(MinutesBeforeRetry+1)*time.Minute))

	retryOccurred := false

	EthWrapper.TransferToken = func(from common.Address, privateKey *ecdsa.PrivateKey, to common.Address,
		opqAmount big.Int, gasPrice *big.Int) (bool, string, int64) {
		retryOccurred = true
		return true, "", 1
	}

	stripePayment.RetryIfTimedOut()

	assert.True(t, retryOccurred)
}

func Test_DeleteStripePaymentIfExists(t *testing.T) {
	DeleteStripePaymentsForTest(t)
	stripePayment := returnValidStripePaymentForTest()

	if err := DB.Create(&stripePayment).Error; err != nil {
		t.Fatalf("should have created row but didn't: " + err.Error())
	}

	accountID := stripePayment.AccountID
	err := DeleteStripePaymentIfExists(accountID)
	assert.Nil(t, err)
	stripeRow, err := GetStripePaymentByAccountId(accountID)
	assert.NotNil(t, err)
	assert.Equal(t, "", stripeRow.StripeToken)
}
