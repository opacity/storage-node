package models

import (
	"encoding/hex"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func returnValidUpgrade() Upgrade {
	account := returnValidAccount()

	// Add account to DB
	DB.Create(&account)

	ethAddress, privateKey, _ := services.EthWrapper.GenerateWallet()

	upgradeCostInOPQ, _ := account.UpgradeCostInOPQ(utils.Env.Plans[1024].StorageInGB,
		12)
	upgradeCostInUSD, _ := account.UpgradeCostInUSD(utils.Env.Plans[1024].StorageInGB,
		12)

	return Upgrade{
		AccountID:        account.AccountID,
		NewStorageLimit:  ProfessionalStorageLimit,
		EthAddress:       ethAddress.String(),
		EthPrivateKey:    hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, account.AccountID)),
		PaymentStatus:    InitialPaymentInProgress,
		OpqCost:          upgradeCostInOPQ,
		UsdCost:          upgradeCostInUSD,
		DurationInMonths: 12,
	}
}

func Test_Init_Upgrades(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.TestDatabaseURL)
}

func Test_Valid_Upgrade_Passes(t *testing.T) {
	upgrade := returnValidUpgrade()

	if err := utils.Validator.Struct(upgrade); err != nil {
		t.Fatalf("upgrade should have passed validation but didn't: " + err.Error())
	}
}

func Test_Upgrade_Empty_AccountID_Fails(t *testing.T) {
	upgrade := returnValidUpgrade()
	upgrade.AccountID = ""

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}
}

func Test_Upgrade_Invalid_AccountID_Length_Fails(t *testing.T) {
	upgrade := returnValidUpgrade()
	upgrade.AccountID = utils.RandSeqFromRunes(63, []rune("abcdef01234567890"))

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}

	upgrade.AccountID = utils.RandSeqFromRunes(65, []rune("abcdef01234567890"))

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}
}

func Test_Upgrade_Not_Enough_Months_Fails(t *testing.T) {
	upgrade := returnValidUpgrade()
	upgrade.DurationInMonths = 0

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}
}

func Test_Upgrade_StorageLimit_Less_Than_128_Fails(t *testing.T) {
	upgrade := returnValidUpgrade()
	upgrade.NewStorageLimit = 127

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}
}

func Test_Upgrade_No_Eth_Address_Fails(t *testing.T) {
	upgrade := returnValidUpgrade()
	upgrade.EthAddress = ""

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}
}

func Test_Upgrade_Eth_Address_Invalid_Length_Fails(t *testing.T) {
	upgrade := returnValidUpgrade()
	upgrade.EthAddress = utils.RandSeqFromRunes(6, []rune("abcdef01234567890"))

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}
}

func Test_Upgrade_No_Eth_Private_Key_Fails(t *testing.T) {
	upgrade := returnValidUpgrade()
	upgrade.EthPrivateKey = ""

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}
}

func Test_Upgrade_Eth_Private_Key_Invalid_Length_Fails(t *testing.T) {
	upgrade := returnValidUpgrade()
	upgrade.EthPrivateKey = utils.RandSeqFromRunes(6, []rune("abcdef01234567890"))

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}
}

func Test_Upgrade_No_Payment_Status_Fails(t *testing.T) {
	upgrade := returnValidUpgrade()
	upgrade.PaymentStatus = 0

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}
}

func Test_Upgrade_GetOrCreateUpgrade(t *testing.T) {
	upgrade := returnValidUpgrade()

	// Test that new upgrade is created
	uPtr, err := GetOrCreateUpgrade(upgrade)
	assert.Nil(t, err)
	assert.Equal(t, uPtr.AccountID, upgrade.AccountID)
	assert.Equal(t, uPtr.EthAddress, upgrade.EthAddress)
	assert.Equal(t, uPtr.NewStorageLimit, upgrade.NewStorageLimit)
	assert.Equal(t, uPtr.OpqCost, upgrade.OpqCost)
	assert.Equal(t, uPtr.UsdCost, upgrade.UsdCost)

	// simulate generating a new update with the same AccountID and NewStorageLimit
	// although another upgrade already exists
	upgrade2 := returnValidUpgrade()
	upgrade2.AccountID = upgrade.AccountID
	upgrade2.NewStorageLimit = upgrade.NewStorageLimit
	upgrade2.OpqCost = 1337.00
	uPtr, err = GetOrCreateUpgrade(upgrade2)
	assert.Nil(t, err)

	// verify AccountID, EthAddress, EthPrivateKey are still the same
	assert.Equal(t, uPtr.AccountID, upgrade.AccountID)
	assert.Equal(t, uPtr.EthAddress, upgrade.EthAddress)
	assert.Equal(t, uPtr.EthPrivateKey, upgrade.EthPrivateKey)

	// verify OpqCost has changed
	assert.Equal(t, upgrade2.OpqCost, uPtr.OpqCost)
	assert.NotEqual(t, upgrade.OpqCost, uPtr.OpqCost)
}

func Test_Upgrade_GetUpgradeFromAccountIDAndNewStorageLimit(t *testing.T) {
	upgrade := returnValidUpgrade()

	DB.Create(&upgrade)

	upgradeFromDB, err := GetUpgradeFromAccountIDAndNewStorageLimit(upgrade.AccountID, int(upgrade.NewStorageLimit))
	assert.Nil(t, err)
	assert.Equal(t, upgrade.AccountID, upgradeFromDB.AccountID)

	upgradeFromDB, err = GetUpgradeFromAccountIDAndNewStorageLimit(
		utils.RandSeqFromRunes(AccountIDLength, []rune("abcdef01234567890")), 128)
	assert.NotNil(t, err)
	assert.Equal(t, "", upgradeFromDB.AccountID)
}

func Test_Upgrade_GetUpgradesFromAccountID(t *testing.T) {
	upgrade := returnValidUpgrade()

	DB.Create(&upgrade)

	upgrade2 := returnValidUpgrade()
	upgrade2.NewStorageLimit = BasicStorageLimit
	upgrade2.AccountID = upgrade.AccountID

	err := DB.Create(&upgrade2).Error
	assert.Nil(t, err)

	upgrades, err := GetUpgradesFromAccountID(upgrade.AccountID)
	assert.Nil(t, err)

	assert.Equal(t, 2, len(upgrades))
}

func Test_Upgrade_SetUpgradesToNextPaymentStatus(t *testing.T) {
	upgrade := returnValidUpgrade()

	DB.Create(&upgrade)

	assert.Equal(t, InitialPaymentInProgress, upgrade.PaymentStatus)

	SetUpgradesToNextPaymentStatus([]Upgrade{upgrade})
	upgradeFromDB, err := GetUpgradeFromAccountIDAndNewStorageLimit(
		upgrade.AccountID, int(upgrade.NewStorageLimit))
	assert.Nil(t, err)
	assert.Equal(t, InitialPaymentReceived, upgradeFromDB.PaymentStatus)

	SetUpgradesToNextPaymentStatus([]Upgrade{upgradeFromDB})
	upgradeFromDB, err = GetUpgradeFromAccountIDAndNewStorageLimit(
		upgrade.AccountID, int(upgrade.NewStorageLimit))
	assert.Nil(t, err)
	assert.Equal(t, GasTransferInProgress, upgradeFromDB.PaymentStatus)

	SetUpgradesToNextPaymentStatus([]Upgrade{upgradeFromDB})
	upgradeFromDB, err = GetUpgradeFromAccountIDAndNewStorageLimit(
		upgrade.AccountID, int(upgrade.NewStorageLimit))
	assert.Nil(t, err)
	assert.Equal(t, GasTransferComplete, upgradeFromDB.PaymentStatus)

	SetUpgradesToNextPaymentStatus([]Upgrade{upgradeFromDB})
	upgradeFromDB, err = GetUpgradeFromAccountIDAndNewStorageLimit(
		upgrade.AccountID, int(upgrade.NewStorageLimit))
	assert.Nil(t, err)
	assert.Equal(t, PaymentRetrievalInProgress, upgradeFromDB.PaymentStatus)

	SetUpgradesToNextPaymentStatus([]Upgrade{upgradeFromDB})
	upgradeFromDB, err = GetUpgradeFromAccountIDAndNewStorageLimit(
		upgrade.AccountID, int(upgrade.NewStorageLimit))
	assert.Nil(t, err)
	assert.Equal(t, PaymentRetrievalComplete, upgradeFromDB.PaymentStatus)

	SetUpgradesToNextPaymentStatus([]Upgrade{upgradeFromDB})
	upgradeFromDB, err = GetUpgradeFromAccountIDAndNewStorageLimit(
		upgrade.AccountID, int(upgrade.NewStorageLimit))
	assert.Nil(t, err)
	assert.Equal(t, PaymentRetrievalComplete, upgradeFromDB.PaymentStatus)
}

func Test_Upgrade_CheckIfPaid_Has_Paid(t *testing.T) {
	upgrade := returnValidUpgrade()

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return true, nil
	}

	if err := DB.Create(&upgrade).Error; err != nil {
		t.Fatalf("should have upgrade account but didn't: " + err.Error())
	}

	paid, err := upgrade.CheckIfPaid()
	assert.True(t, paid)
	assert.Nil(t, err)

	upgradesFromDB, _ := GetUpgradesFromAccountID(upgrade.AccountID)

	assert.Equal(t, InitialPaymentReceived, upgradesFromDB[0].PaymentStatus)
}

func Test_Upgrade_CheckIfPaid_Not_Paid(t *testing.T) {
	upgrade := returnValidUpgrade()

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	if err := DB.Create(&upgrade).Error; err != nil {
		t.Fatalf("should have upgrade account but didn't: " + err.Error())
	}

	paid, err := upgrade.CheckIfPaid()
	assert.False(t, paid)
	assert.Nil(t, err)

	upgradesFromDB, _ := GetUpgradesFromAccountID(upgrade.AccountID)

	assert.Equal(t, InitialPaymentInProgress, upgradesFromDB[0].PaymentStatus)
}

func Test_Upgrade_CheckIfPaid_Error_While_Checking(t *testing.T) {
	upgrade := returnValidUpgrade()

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, errors.New("some error")
	}

	if err := DB.Create(&upgrade).Error; err != nil {
		t.Fatalf("should have upgrade account but didn't: " + err.Error())
	}

	paid, err := upgrade.CheckIfPaid()
	assert.False(t, paid)
	assert.NotNil(t, err)

	upgradesFromDB, _ := GetUpgradesFromAccountID(upgrade.AccountID)

	assert.Equal(t, InitialPaymentInProgress, upgradesFromDB[0].PaymentStatus)
}

func Test_Upgrade_GetTotalCostInWei(t *testing.T) {
	upgrade := returnValidUpgrade()

	costInWei := upgrade.GetTotalCostInWei()

	assert.Equal(t, "15000000000000000000", costInWei.String())
}
