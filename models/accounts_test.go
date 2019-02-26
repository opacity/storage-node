package models

import (
	"errors"
	"testing"

	"encoding/hex"

	"time"

	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

var testNonce = "23384a8eabc4a4ba091cfdbcb3dbacdc27000c03e318fd52accb8e2380f11320"

func returnValidAccount() Account {
	ethAddress, privateKey, _ := services.EthWrapper.GenerateWallet()

	return Account{
		AccountID:            utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
		MonthsInSubscription: 12,
		StorageLocation:      "https://someFileStoragePlace.com/12345",
		StorageLimit:         BasicStorageLimit,
		StorageUsed:          10,
		PaymentStatus:        InitialPaymentInProgress,
		EthAddress:           ethAddress.String(),
		EthPrivateKey:        hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, testNonce)),
	}
}

func Test_Init_Accounts(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.DatabaseURL)
}

func Test_Valid_Account_Passes(t *testing.T) {
	account := returnValidAccount()

	if err := utils.Validator.Struct(account); err != nil {
		t.Fatalf("account should have passed validation but didn't: " + err.Error())
	}
}

func Test_Empty_AccountID_Fails(t *testing.T) {
	account := returnValidAccount()
	account.AccountID = ""

	if err := utils.Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_Invalid_AccountID_Length_Fails(t *testing.T) {
	account := returnValidAccount()
	account.AccountID = utils.RandSeqFromRunes(63, []rune("abcdef01234567890"))

	if err := utils.Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}

	account.AccountID = utils.RandSeqFromRunes(65, []rune("abcdef01234567890"))

	if err := utils.Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_Not_Enough_Months_Fails(t *testing.T) {
	account := returnValidAccount()
	account.MonthsInSubscription = 0

	if err := utils.Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_StorageLocation_Invalid_URL_Fails(t *testing.T) {
	account := returnValidAccount()
	account.StorageLocation = "wrong"

	if err := utils.Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_StorageLimit_Less_Than_100_Fails(t *testing.T) {
	account := returnValidAccount()
	account.StorageLimit = 99

	if err := utils.Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_No_Eth_Address_Fails(t *testing.T) {
	account := returnValidAccount()
	account.EthAddress = ""

	if err := utils.Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_Eth_Address_Invalid_Length_Fails(t *testing.T) {
	account := returnValidAccount()
	account.EthAddress = utils.RandSeqFromRunes(6, []rune("abcdef01234567890"))

	if err := utils.Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_No_Eth_Private_Key_Fails(t *testing.T) {
	account := returnValidAccount()
	account.EthPrivateKey = ""

	if err := utils.Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_Eth_Private_Key_Invalid_Length_Fails(t *testing.T) {
	account := returnValidAccount()
	account.EthPrivateKey = utils.RandSeqFromRunes(6, []rune("abcdef01234567890"))

	if err := utils.Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_No_Payment_Status_Fails(t *testing.T) {
	account := returnValidAccount()
	account.PaymentStatus = 0

	if err := utils.Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_Returns_Expiration_Date(t *testing.T) {
	account := returnValidAccount()
	account.MonthsInSubscription = 24

	if err := utils.Validator.Struct(account); err != nil {
		t.Fatalf("account should have passed validation")
	}

	// Add account to DB
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	currentTime := time.Now()
	expirationDate := account.ExpirationDate()

	assert.Equal(t, currentTime.Year()+2, expirationDate.Year())
	assert.Equal(t, currentTime.Month(), expirationDate.Month())
}

func Test_Returns_Cost(t *testing.T) {
	account := returnValidAccount()
	account.MonthsInSubscription = DefaultMonthsPerSubscription

	if err := utils.Validator.Struct(account); err != nil {
		t.Fatalf("account should have passed validation")
	}

	cost, err := account.Cost()

	if err != nil {
		t.Fatalf("should have been able to calculate cost")
	}

	assert.Equal(t, BasicSubscriptionDefaultCost, cost)
}

func Test_GetTotalCostInWei(t *testing.T) {
	account := returnValidAccount()
	account.MonthsInSubscription = DefaultMonthsPerSubscription

	if err := utils.Validator.Struct(account); err != nil {
		t.Fatalf("account should have passed validation")
	}

	costInWei := account.GetTotalCostInWei()

	assert.Equal(t, big.NewInt(1560000000000000000).String(), costInWei.String())
}

func Test_CheckIfPaid_Has_Paid(t *testing.T) {
	account := returnValidAccount()
	account.MonthsInSubscription = DefaultMonthsPerSubscription

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return true, nil
	}

	paid, err := account.CheckIfPaid()
	assert.True(t, paid)
	assert.Nil(t, err)
	assert.Equal(t, InitialPaymentReceived, account.PaymentStatus)
}

func Test_CheckIfPaid_Not_Paid(t *testing.T) {
	account := returnValidAccount()
	account.MonthsInSubscription = DefaultMonthsPerSubscription

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	paid, err := account.CheckIfPaid()
	assert.False(t, paid)
	assert.Nil(t, err)
	assert.Equal(t, InitialPaymentInProgress, account.PaymentStatus)
}

func Test_CheckIfPaid_Error_While_Checking(t *testing.T) {
	account := returnValidAccount()
	account.MonthsInSubscription = DefaultMonthsPerSubscription

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, errors.New("some error")
	}

	paid, err := account.CheckIfPaid()
	assert.False(t, paid)
	assert.NotNil(t, err)
	assert.Equal(t, InitialPaymentInProgress, account.PaymentStatus)
}

func Test_CreateAndGet_Account(t *testing.T) {
	account := returnValidAccount()
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	savedAccount, err := GetAccountById(account.AccountID)

	assert.Nil(t, err)

	// Time maybe different since one is before saved. Just ignore the time field difference.
	account.CreatedAt = savedAccount.CreatedAt
	account.UpdatedAt = savedAccount.UpdatedAt
	assert.Equal(t, account, savedAccount)
}

func Test_HasEnoughSpaceToUploadFile(t *testing.T) {
	account := returnValidAccount()
	account.PaymentStatus = PaymentRetrievalComplete
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	assert.Nil(t, account.UseStorageSpaceInByte(10*1e9 /* Upload 10GB. */))
}

func Test_NoEnoughSpaceToUploadFile(t *testing.T) {
	account := returnValidAccount()
	account.PaymentStatus = PaymentRetrievalComplete
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	assert.NotNil(t, account.UseStorageSpaceInByte(95*1e9 /* Upload 95GB. */))
}
