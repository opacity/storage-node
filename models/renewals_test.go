package models

import (
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func returnValidRenewal() (Renewal, Account) {
	account := returnValidAccount()

	// Add account to DB
	DB.Create(&account)

	ethAddress, privateKey := services.GenerateWallet()

	renewalCostInOPCT, _ := account.Cost()

	return Renewal{
		AccountID:     account.AccountID,
		EthAddress:    ethAddress.String(),
		EthPrivateKey: hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, account.AccountID)),
		PaymentStatus: InitialPaymentInProgress,
		OpctCost:      renewalCostInOPCT,
		//UsdCost:          utils.Env.Plans[int(account.StorageLimit)].CostInUSD,
		DurationInMonths: 12,
		NetworkIdPaid:    utils.TestNetworkID,
	}, account
}

func Test_Init_Renewals(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.TestDatabaseURL)
}

func Test_Valid_Renewal_Passes(t *testing.T) {
	renewal, _ := returnValidRenewal()

	if err := utils.Validator.Struct(renewal); err != nil {
		t.Fatalf("renewal should have passed validation but didn't: " + err.Error())
	}
}

func Test_Renewal_Empty_AccountID_Fails(t *testing.T) {
	renewal, _ := returnValidRenewal()
	renewal.AccountID = ""

	if err := utils.Validator.Struct(renewal); err == nil {
		t.Fatalf("renewal should have failed validation")
	}
}

func Test_Renewal_Invalid_AccountID_Length_Fails(t *testing.T) {
	renewal, _ := returnValidRenewal()
	renewal.AccountID = utils.RandSeqFromRunes(63, []rune("abcdef01234567890"))

	if err := utils.Validator.Struct(renewal); err == nil {
		t.Fatalf("renewal should have failed validation")
	}

	renewal.AccountID = utils.RandSeqFromRunes(65, []rune("abcdef01234567890"))

	if err := utils.Validator.Struct(renewal); err == nil {
		t.Fatalf("renewal should have failed validation")
	}
}

func Test_Renewal_Not_Enough_Months_Fails(t *testing.T) {
	renewal, _ := returnValidRenewal()
	renewal.DurationInMonths = 0

	if err := utils.Validator.Struct(renewal); err == nil {
		t.Fatalf("renewal should have failed validation")
	}
}

func Test_Renewal_No_Eth_Address_Fails(t *testing.T) {
	renewal, _ := returnValidRenewal()
	renewal.EthAddress = ""

	if err := utils.Validator.Struct(renewal); err == nil {
		t.Fatalf("renewal should have failed validation")
	}
}

func Test_Renewal_Eth_Address_Invalid_Length_Fails(t *testing.T) {
	renewal, _ := returnValidRenewal()
	renewal.EthAddress = utils.RandSeqFromRunes(6, []rune("abcdef01234567890"))

	if err := utils.Validator.Struct(renewal); err == nil {
		t.Fatalf("renewal should have failed validation")
	}
}

func Test_Renewal_No_Eth_Private_Key_Fails(t *testing.T) {
	renewal, _ := returnValidRenewal()
	renewal.EthPrivateKey = ""

	if err := utils.Validator.Struct(renewal); err == nil {
		t.Fatalf("renewal should have failed validation")
	}
}

func Test_Renewal_Eth_Private_Key_Invalid_Length_Fails(t *testing.T) {
	renewal, _ := returnValidRenewal()
	renewal.EthPrivateKey = utils.RandSeqFromRunes(6, []rune("abcdef01234567890"))

	if err := utils.Validator.Struct(renewal); err == nil {
		t.Fatalf("renewal should have failed validation")
	}
}

func Test_Renewal_No_Payment_Status_Fails(t *testing.T) {
	renewal, _ := returnValidRenewal()
	renewal.PaymentStatus = 0

	if err := utils.Validator.Struct(renewal); err == nil {
		t.Fatalf("renewal should have failed validation")
	}
}

func Test_Renewal_GetOrCreateRenewal(t *testing.T) {
	renewal, _ := returnValidRenewal()

	// Test that new renewal is created
	uPtr, err := GetOrCreateRenewal(renewal)
	assert.Nil(t, err)
	assert.Equal(t, uPtr.AccountID, renewal.AccountID)
	assert.Equal(t, uPtr.EthAddress, renewal.EthAddress)
	assert.Equal(t, uPtr.OpctCost, renewal.OpctCost)
	//assert.Equal(t, uPtr.UsdCost, renewal.UsdCost)

	// simulate generating a new update with the same AccountID
	// although another renewal already exists--price should not change
	renewal2, _ := returnValidRenewal()
	renewal2.AccountID = renewal.AccountID
	renewal2.OpctCost = 1337.00
	uPtr, err = GetOrCreateRenewal(renewal2)
	assert.Nil(t, err)

	// verify AccountID, EthAddress, EthPrivateKey are still the same
	assert.Equal(t, uPtr.AccountID, renewal.AccountID)
	assert.Equal(t, uPtr.EthAddress, renewal.EthAddress)
	assert.Equal(t, uPtr.EthPrivateKey, renewal.EthPrivateKey)

	// verify OpctCost has NOT changed -- there was already a renewal in the DB so we returned the one we already had
	assert.NotEqual(t, renewal2.OpctCost, uPtr.OpctCost)
	assert.Equal(t, renewal.OpctCost, uPtr.OpctCost)
}

func Test_Renewal_GetRenewalsFromAccountID(t *testing.T) {
	renewal, _ := returnValidRenewal()

	DB.Create(&renewal)

	renewal2, _ := returnValidRenewal()
	renewal2.AccountID = renewal.AccountID

	err := DB.Create(&renewal2).Error
	assert.Error(t, err)

	renewals, err := GetRenewalsFromAccountID(renewal.AccountID)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(renewals))
}

func Test_Renewal_SetRenewalsToNextPaymentStatus(t *testing.T) {
	renewal, account := returnValidRenewal()

	DB.Create(&renewal)

	assert.Equal(t, InitialPaymentInProgress, renewal.PaymentStatus)

	SetRenewalsToNextPaymentStatus([]Renewal{renewal})
	renewalFromDB, err := GetRenewalsFromAccountID(account.AccountID)
	assert.Nil(t, err)
	assert.Equal(t, InitialPaymentReceived, renewalFromDB[0].PaymentStatus)

	SetRenewalsToNextPaymentStatus([]Renewal{renewalFromDB[0]})
	renewalFromDB, err = GetRenewalsFromAccountID(account.AccountID)
	assert.Nil(t, err)
	assert.Equal(t, GasTransferInProgress, renewalFromDB[0].PaymentStatus)

	SetRenewalsToNextPaymentStatus([]Renewal{renewalFromDB[0]})
	renewalFromDB, err = GetRenewalsFromAccountID(account.AccountID)
	assert.Nil(t, err)
	assert.Equal(t, GasTransferComplete, renewalFromDB[0].PaymentStatus)

	SetRenewalsToNextPaymentStatus([]Renewal{renewalFromDB[0]})
	renewalFromDB, err = GetRenewalsFromAccountID(account.AccountID)
	assert.Nil(t, err)
	assert.Equal(t, PaymentRetrievalInProgress, renewalFromDB[0].PaymentStatus)

	SetRenewalsToNextPaymentStatus([]Renewal{renewalFromDB[0]})
	renewalFromDB, err = GetRenewalsFromAccountID(account.AccountID)
	assert.Nil(t, err)
	assert.Equal(t, PaymentRetrievalComplete, renewalFromDB[0].PaymentStatus)

	SetRenewalsToNextPaymentStatus([]Renewal{renewalFromDB[0]})
	renewalFromDB, err = GetRenewalsFromAccountID(account.AccountID)
	assert.Nil(t, err)
	assert.Equal(t, PaymentRetrievalComplete, renewalFromDB[0].PaymentStatus)
}

func Test_Renewal_CheckIfPaid_Has_Paid(t *testing.T) {
	renewal, _ := returnValidRenewal()

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return true, utils.TestNetworkID, nil
	}

	if err := DB.Create(&renewal).Error; err != nil {
		t.Fatalf("should have renewal account but didn't: " + err.Error())
	}

	paid, _, err := renewal.CheckIfPaid()
	assert.True(t, paid)
	assert.Nil(t, err)

	renewalsFromDB, _ := GetRenewalsFromAccountID(renewal.AccountID)

	assert.Equal(t, InitialPaymentReceived, renewalsFromDB[0].PaymentStatus)
}

func Test_Renewal_CheckIfPaid_Not_Paid(t *testing.T) {
	renewal, _ := returnValidRenewal()

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return false, utils.TestNetworkID, nil
	}

	if err := DB.Create(&renewal).Error; err != nil {
		t.Fatalf("should have renewal account but didn't: " + err.Error())
	}

	paid, _, err := renewal.CheckIfPaid()
	assert.False(t, paid)
	assert.Nil(t, err)

	renewalsFromDB, _ := GetRenewalsFromAccountID(renewal.AccountID)

	assert.Equal(t, InitialPaymentInProgress, renewalsFromDB[0].PaymentStatus)
}

func Test_Renewal_CheckIfPaid_Error_While_Checking(t *testing.T) {
	renewal, _ := returnValidRenewal()

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return false, 0, errors.New("some error")
	}

	if err := DB.Create(&renewal).Error; err != nil {
		t.Fatalf("should have renewal account but didn't: " + err.Error())
	}

	paid, _, err := renewal.CheckIfPaid()
	assert.False(t, paid)
	assert.NotNil(t, err)

	renewalsFromDB, _ := GetRenewalsFromAccountID(renewal.AccountID)

	assert.Equal(t, InitialPaymentInProgress, renewalsFromDB[0].PaymentStatus)
}

func Test_Renewal_GetTotalCostInWei(t *testing.T) {
	renewal, _ := returnValidRenewal()

	costInWei := renewal.GetTotalCostInWei()

	assert.Equal(t, "2000000000000000000", costInWei.String())
}

func Test_SetRenewalsToLowerPaymentStatusByUpdateTime(t *testing.T) {
	DeleteRenewalsForTest(t)

	for i := 0; i < 2; i++ {
		renewal, _ := returnValidRenewal()
		if err := DB.Create(&renewal).Error; err != nil {
			t.Fatalf("should have created renewal but didn't: " + err.Error())
		}
	}

	renewals := []Renewal{}
	DB.Find(&renewals)
	assert.Equal(t, 2, len(renewals))

	renewals[0].PaymentStatus = GasTransferInProgress
	renewals[1].PaymentStatus = GasTransferInProgress

	DB.Save(&renewals[0])
	DB.Save(&renewals[1])

	// after cutoff time
	// should NOT get set to lower status
	DB.Exec("UPDATE renewals set updated_at = ? WHERE account_id = ?;", time.Now().Add(-1*1*24*time.Hour), renewals[0].AccountID)
	// before cutoff time
	// should get set to lower status
	DB.Exec("UPDATE renewals set updated_at = ? WHERE account_id = ?;", time.Now().Add(-1*3*24*time.Hour), renewals[1].AccountID)

	err := SetRenewalsToLowerPaymentStatusByUpdateTime(GasTransferInProgress, time.Now().Add(-1*2*24*time.Hour))
	assert.Nil(t, err)

	renewalsFromDB := []Renewal{}
	DB.Find(&renewalsFromDB)

	if renewalsFromDB[0].AccountID == renewals[0].AccountID {
		assert.Equal(t, GasTransferInProgress, renewalsFromDB[0].PaymentStatus)
	}

	if renewalsFromDB[0].AccountID == renewals[1].AccountID {
		assert.Equal(t, InitialPaymentReceived, renewalsFromDB[0].PaymentStatus)
	}

	if renewalsFromDB[1].AccountID == renewals[0].AccountID {
		assert.Equal(t, GasTransferInProgress, renewalsFromDB[1].PaymentStatus)
	}

	if renewalsFromDB[1].AccountID == renewals[1].AccountID {
		assert.Equal(t, InitialPaymentReceived, renewalsFromDB[1].PaymentStatus)
	}
}
