package models

import (
	"errors"
	"testing"

	"encoding/hex"

	"time"

	"math/big"

	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func returnValidAccount() Account {
	ethAddress, privateKey, _ := services.EthWrapper.GenerateWallet()
	accountID := utils.RandSeqFromRunes(64, []rune("abcdef01234567890"))

	return Account{
		AccountID:            accountID,
		MonthsInSubscription: DefaultMonthsPerSubscription,
		StorageLocation:      "https://someFileStoragePlace.com/12345",
		StorageLimit:         BasicStorageLimit,
		StorageUsed:          10,
		PaymentStatus:        InitialPaymentInProgress,
		EthAddress:           ethAddress.String(),
		EthPrivateKey:        hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, accountID)),
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

	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	paid, err := account.CheckIfPaid()
	assert.True(t, paid)
	assert.Nil(t, err)

	accountFromDB, _ := GetAccountById(account.AccountID)

	assert.Equal(t, InitialPaymentReceived, accountFromDB.PaymentStatus)
}

func Test_CheckIfPaid_Not_Paid(t *testing.T) {
	account := returnValidAccount()
	account.MonthsInSubscription = DefaultMonthsPerSubscription

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	paid, err := account.CheckIfPaid()
	assert.False(t, paid)
	assert.Nil(t, err)

	accountFromDB, _ := GetAccountById(account.AccountID)

	assert.Equal(t, InitialPaymentInProgress, accountFromDB.PaymentStatus)
}

func Test_CheckIfPaid_Error_While_Checking(t *testing.T) {
	account := returnValidAccount()
	account.MonthsInSubscription = DefaultMonthsPerSubscription

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, errors.New("some error")
	}

	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	paid, err := account.CheckIfPaid()
	assert.False(t, paid)
	assert.NotNil(t, err)

	accountFromDB, _ := GetAccountById(account.AccountID)

	assert.Equal(t, InitialPaymentInProgress, accountFromDB.PaymentStatus)
}

func Test_CheckIfPending_Is_Pending(t *testing.T) {
	account := returnValidAccount()

	BackendManager.CheckIfPending = func(address common.Address) (bool, error) {
		return true, nil
	}

	pending, err := account.CheckIfPending()
	assert.True(t, pending)
	assert.Nil(t, err)
}

func Test_CheckIfPending_Is_Not_Pending(t *testing.T) {
	account := returnValidAccount()

	BackendManager.CheckIfPending = func(address common.Address) (bool, error) {
		return false, nil
	}

	pending, err := account.CheckIfPending()
	assert.False(t, pending)
	assert.Nil(t, err)
}

func Test_CheckIfPending_Error_While_Checking(t *testing.T) {
	account := returnValidAccount()

	BackendManager.CheckIfPending = func(address common.Address) (bool, error) {
		return false, errors.New("some error")
	}

	pending, err := account.CheckIfPending()
	assert.False(t, pending)
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

func Test_CreateSpaceUsedReport(t *testing.T) {
	expectedSpaceAlloted := 400
	expectedSpaceUsed := 234.56

	if err := DB.Delete(&Account{}).Error; err != nil {
		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
	}

	for i := 0; i < 4; i++ {
		accountPaid := returnValidAccount()
		accountPaid.StorageUsed = expectedSpaceUsed / 4
		accountPaid.PaymentStatus = PaymentStatusType(utils.RandIndex(5) + 2)
		if err := DB.Create(&accountPaid).Error; err != nil {
			t.Fatalf("should have created account but didn't: " + err.Error())
		}
	}

	for i := 0; i < 4; i++ {
		accountUnpaid := returnValidAccount()
		accountUnpaid.StorageUsed = expectedSpaceUsed / 4
		if err := DB.Create(&accountUnpaid).Error; err != nil {
			t.Fatalf("should have created account but didn't: " + err.Error())
		}
	}

	spaceReport := CreateSpaceUsedReport()

	assert.Equal(t, expectedSpaceAlloted, spaceReport.SpaceAllotedSum)
	assert.Equal(t, expectedSpaceUsed, spaceReport.SpaceUsedSum)
}

func Test_PurgeOldUnpaidAccounts(t *testing.T) {
	if err := DB.Delete(&Account{}).Error; err != nil {
		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
	}

	for i := 0; i < 4; i++ {
		accountPaid := returnValidAccount()
		if err := DB.Create(&accountPaid).Error; err != nil {
			t.Fatalf("should have created account but didn't: " + err.Error())
		}
	}

	accounts := []Account{}
	DB.Find(&accounts)
	assert.Equal(t, 4, len(accounts))

	// after cutoff time and payment has been received
	// should NOT get purged
	accounts[0].CreatedAt = time.Now().Add(-1 * 6 * 24 * time.Hour)
	accounts[0].PaymentStatus = InitialPaymentReceived

	// before cutoff time but payment has been received
	// should NOT get purged
	accounts[1].CreatedAt = time.Now().Add(-1 * 8 * 24 * time.Hour)
	accounts[1].PaymentStatus = InitialPaymentReceived

	// after cutoff time, payment still in progress
	// should NOT get purged
	accounts[2].CreatedAt = time.Now().Add(-1 * 6 * 24 * time.Hour)
	accounts[2].PaymentStatus = InitialPaymentInProgress

	// before cutoff time, payment still in progress
	// this one should get purged
	accounts[3].CreatedAt = time.Now().Add(-1 * 8 * 24 * time.Hour)
	accounts[3].PaymentStatus = InitialPaymentInProgress

	accountToBeDeletedID := accounts[3].AccountID

	DB.Save(&accounts[0])
	DB.Save(&accounts[1])
	DB.Save(&accounts[2])
	DB.Save(&accounts[3])

	PurgeOldUnpaidAccounts(7)

	accounts = []Account{}
	DB.Find(&accounts)
	assert.Equal(t, 3, len(accounts))

	accounts = []Account{}
	DB.Where("account_id = ?", accountToBeDeletedID).Find(&accounts)
	assert.Equal(t, 0, len(accounts))
}

func Test_GetAccountsByPaymentStatus(t *testing.T) {
	if err := DB.Delete(&Account{}).Error; err != nil {
		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
	}

	// for each payment status, check that we can get the accounts of that status and that the account IDs
	// of the accounts returned from GetAccountsByPaymentStatus match the accounts we created for the test
	for paymentStatus := InitialPaymentInProgress; paymentStatus <= PaymentRetrievalComplete; paymentStatus++ {
		account := returnValidAccount()
		account.PaymentStatus = paymentStatus
		if err := DB.Create(&account).Error; err != nil {
			t.Fatalf("should have created account but didn't: " + err.Error())
		}
		expectedAccountID := account.AccountID
		accounts := GetAccountsByPaymentStatus(paymentStatus)
		assert.Equal(t, 1, len(accounts))
		assert.Equal(t, expectedAccountID, accounts[0].AccountID)
	}
}

func Test_SetAccountsToNextPaymentStatus(t *testing.T) {
	for paymentStatus := InitialPaymentInProgress; paymentStatus <= PaymentRetrievalComplete; paymentStatus++ {
		if err := DB.Delete(&Account{}).Error; err != nil {
			t.Fatalf("should have deleted accounts but didn't: " + err.Error())
		}
		account := returnValidAccount()

		// set to starting payment status
		account.PaymentStatus = paymentStatus
		if err := DB.Create(&account).Error; err != nil {
			t.Fatalf("should have created account but didn't: " + err.Error())
		}
		accounts := GetAccountsByPaymentStatus(paymentStatus)

		// verify 1 account was returned and that its payment status is as we expected
		assert.Equal(t, 1, len(accounts))
		assert.Equal(t, paymentStatus, accounts[0].PaymentStatus)

		// call method under test
		SetAccountsToNextPaymentStatus(accounts)

		// get the next status in the sequence, or keep the same status if the it was the
		// last status in the sequence
		newStatus := getNextPaymentStatus(paymentStatus)

		// verify that the starting paymentStatus and newStatus are not equal, UNLESS
		// they are both the last status in the sequence
		assert.True(t, newStatus != paymentStatus ||
			(newStatus == paymentStatus && paymentStatus == PaymentRetrievalComplete))

		// if the payment statuses are not equal verify there are no accounts returned
		// with the original payment status
		if paymentStatus != newStatus {
			accounts = GetAccountsByPaymentStatus(paymentStatus)
			assert.Equal(t, 0, len(accounts))
		}

		// verify there is 1 account with the new status
		accounts = GetAccountsByPaymentStatus(newStatus)
		assert.Equal(t, 1, len(accounts))
	}
}

func Test_handleAccountWithPaymentInProgress_has_paid(t *testing.T) {
	if err := DB.Delete(&Account{}).Error; err != nil {
		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
	}
	account := returnValidAccount()
	account.PaymentStatus = InitialPaymentInProgress
	account.MetadataKey = utils.RandSeqFromRunes(64, []rune("abcdef01234567890"))
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return true, nil
	}

	// grab the account from the DB
	accountFromDB, _ := GetAccountById(account.AccountID)

	// verify no badger pair exists with that metadata key
	_, _, err := utils.GetValueFromKV(accountFromDB.MetadataKey)
	assert.NotNil(t, err)

	// verify that the account has a metadata key
	assert.Equal(t, 64, len(accountFromDB.MetadataKey))

	verifyPaymentStatusExpectations(t, account, InitialPaymentInProgress, InitialPaymentReceived, handleAccountWithPaymentInProgress)

	// The user has paid so we expect changes after calling handleAccountWithPaymentInProgress

	// grab the account from the DB
	accountFromDB, _ = GetAccountById(account.AccountID)

	// verify a badger pair exists with that metadata key
	metadata, _, err := utils.GetValueFromKV(account.MetadataKey)
	assert.Nil(t, err)
	assert.Equal(t, "", metadata)

	//verify account metadata key is deleted
	assert.Equal(t, 0, len(accountFromDB.MetadataKey))
}

func Test_handleAccountWithPaymentInProgress_has_not_paid(t *testing.T) {
	if err := DB.Delete(&Account{}).Error; err != nil {
		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
	}
	account := returnValidAccount()
	account.PaymentStatus = InitialPaymentInProgress
	account.MetadataKey = utils.RandSeqFromRunes(64, []rune("abcdef01234567890"))
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	// grab the account from the DB
	accountFromDB, _ := GetAccountById(account.AccountID)

	// verify no badger pair exists with that metadata key
	_, _, err := utils.GetValueFromKV(accountFromDB.MetadataKey)
	assert.NotNil(t, err)

	// verify that the account has a metadata key
	assert.Equal(t, 64, len(accountFromDB.MetadataKey))

	verifyPaymentStatusExpectations(t, account, InitialPaymentInProgress, InitialPaymentInProgress, handleAccountWithPaymentInProgress)

	// The user has not paid so we expect everything to be the same

	// grab the account from the DB
	accountFromDB, _ = GetAccountById(account.AccountID)

	// verify no badger pair exists with that metadata key
	_, _, err = utils.GetValueFromKV(accountFromDB.MetadataKey)
	assert.NotNil(t, err)

	// verify that the account has a metadata key
	assert.Equal(t, 64, len(accountFromDB.MetadataKey))
}

func Test_handleAccountThatNeedsGas_transfer_success(t *testing.T) {
	if err := DB.Delete(&Account{}).Error; err != nil {
		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
	}
	account := returnValidAccount()
	account.PaymentStatus = InitialPaymentReceived
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	EthWrapper.TransferETH = func(fromAddress common.Address, fromPrivKey *ecdsa.PrivateKey,
		toAddr common.Address, amount *big.Int) (types.Transactions, string, int64, error) {
		// not returning anything important for the first three return values because
		// handleAccountThatNeedsGas only cares about the 4th return value which will
		// be an error or nil
		return types.Transactions{}, "", -1, nil
	}

	verifyPaymentStatusExpectations(t, account, InitialPaymentReceived, GasTransferInProgress, handleAccountThatNeedsGas)

}

func Test_handleAccountThatNeedsGas_transfer_error(t *testing.T) {
	if err := DB.Delete(&Account{}).Error; err != nil {
		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
	}
	account := returnValidAccount()
	account.PaymentStatus = InitialPaymentReceived
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	EthWrapper.TransferETH = func(fromAddress common.Address, fromPrivKey *ecdsa.PrivateKey,
		toAddr common.Address, amount *big.Int) (types.Transactions, string, int64, error) {
		// not returning anything important for the first three return values because
		// handleAccountThatNeedsGas only cares about the 4th return value which will
		// be an error or nil
		return types.Transactions{}, "", -1, errors.New("SOMETHING HAPPENED")
	}

	verifyPaymentStatusExpectations(t, account, InitialPaymentReceived, InitialPaymentReceived, handleAccountThatNeedsGas)

}

func Test_handleAccountReceivingGas_gas_received(t *testing.T) {
	if err := DB.Delete(&Account{}).Error; err != nil {
		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
	}
	account := returnValidAccount()
	account.PaymentStatus = GasTransferInProgress
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	EthWrapper.GetETHBalance = func(addr common.Address) *big.Int {
		return big.NewInt(1)
	}

	verifyPaymentStatusExpectations(t, account, GasTransferInProgress, GasTransferComplete, handleAccountReceivingGas)

}

func Test_handleAccountReceivingGas_gas_not_received(t *testing.T) {
	if err := DB.Delete(&Account{}).Error; err != nil {
		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
	}
	account := returnValidAccount()
	account.PaymentStatus = GasTransferInProgress
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	EthWrapper.GetETHBalance = func(addr common.Address) *big.Int {
		return big.NewInt(-1)
	}

	verifyPaymentStatusExpectations(t, account, GasTransferInProgress, GasTransferInProgress, handleAccountReceivingGas)

}

func Test_handleAccountReadyForCollection_transfer_success(t *testing.T) {
	if err := DB.Delete(&Account{}).Error; err != nil {
		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
	}
	account := returnValidAccount()
	account.PaymentStatus = GasTransferComplete
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	EthWrapper.GetETHBalance = func(addr common.Address) *big.Int {
		return big.NewInt(1)
	}
	EthWrapper.GetTokenBalance = func(addr common.Address) *big.Int {
		return big.NewInt(1)
	}
	EthWrapper.TransferToken = func(from common.Address, privateKey *ecdsa.PrivateKey, to common.Address,
		opqAmount big.Int) (bool, string, int64) {
		// all that handleAccountReadyForCollection cares about is the first return value
		return true, "", 1
	}

	verifyPaymentStatusExpectations(t, account, GasTransferComplete, PaymentRetrievalInProgress, handleAccountReadyForCollection)

}

func Test_handleAccountReadyForCollection_transfer_failed(t *testing.T) {
	if err := DB.Delete(&Account{}).Error; err != nil {
		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
	}
	account := returnValidAccount()
	account.PaymentStatus = GasTransferComplete
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	EthWrapper.GetETHBalance = func(addr common.Address) *big.Int {
		return big.NewInt(1)
	}
	EthWrapper.GetTokenBalance = func(addr common.Address) *big.Int {
		return big.NewInt(1)
	}
	EthWrapper.TransferToken = func(from common.Address, privateKey *ecdsa.PrivateKey, to common.Address,
		opqAmount big.Int) (bool, string, int64) {
		// all that handleAccountReadyForCollection cares about is the first return value
		return false, "", 1
	}

	verifyPaymentStatusExpectations(t, account, GasTransferComplete, GasTransferComplete, handleAccountReadyForCollection)

}

func Test_handleAccountWithCollectionInProgress_balance_found(t *testing.T) {
	if err := DB.Delete(&Account{}).Error; err != nil {
		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
	}
	account := returnValidAccount()
	account.PaymentStatus = PaymentRetrievalInProgress
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	EthWrapper.GetTokenBalance = func(addr common.Address) *big.Int {
		return big.NewInt(1)
	}

	verifyPaymentStatusExpectations(t, account, PaymentRetrievalInProgress, PaymentRetrievalComplete, handleAccountWithCollectionInProgress)
}

func Test_handleAccountWithCollectionInProgress_balance_not_found(t *testing.T) {
	if err := DB.Delete(&Account{}).Error; err != nil {
		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
	}
	account := returnValidAccount()
	account.PaymentStatus = PaymentRetrievalInProgress
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	EthWrapper.GetTokenBalance = func(addr common.Address) *big.Int {
		return big.NewInt(0)
	}

	verifyPaymentStatusExpectations(t, account, PaymentRetrievalInProgress, PaymentRetrievalInProgress, handleAccountWithCollectionInProgress)
}

func verifyPaymentStatusExpectations(t *testing.T,
	account Account,
	startingStatus PaymentStatusType,
	endingStatus PaymentStatusType,
	methodUnderTest func(Account) error) {
	// grab the account from the DB
	accountFromDB, _ := GetAccountById(account.AccountID)

	// verify account's payment status is what we expect
	assert.Equal(t, startingStatus, accountFromDB.PaymentStatus)

	// call method under test
	methodUnderTest(accountFromDB)

	// grab the account from the DB
	accountFromDB, _ = GetAccountById(account.AccountID)

	// verify account's payment status is what we expect
	assert.Equal(t, endingStatus, accountFromDB.PaymentStatus)
}
