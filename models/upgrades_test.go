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

func returnValidUpgrade() (Upgrade, Account) {
	account := returnValidAccount()

	// Add account to DB
	DB.Create(&account)

	ethAddress, privateKey := services.GenerateWallet()

	upgradeCostInOPCT, _ := account.UpgradeCostInOPCT(utils.Env.Plans[1024].StorageInGB,
		12)
	//upgradeCostInUSD, _ := account.UpgradeCostInUSD(utils.Env.Plans[1024].StorageInGB,
	//	12)

	return Upgrade{
		AccountID:       account.AccountID,
		NewStorageLimit: ProfessionalStorageLimit,
		OldStorageLimit: account.StorageLimit,
		EthAddress:      ethAddress.String(),
		EthPrivateKey:   hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, account.AccountID)),
		PaymentStatus:   InitialPaymentInProgress,
		OpctCost:        upgradeCostInOPCT,
		//UsdCost:          upgradeCostInUSD,
		DurationInMonths: 12,
		NetworkIdPaid:    utils.TestNetworkID,
	}, account
}

func Test_Init_Upgrades(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.TestDatabaseURL)
}

func Test_Valid_Upgrade_Passes(t *testing.T) {
	upgrade, _ := returnValidUpgrade()

	if err := utils.Validator.Struct(upgrade); err != nil {
		t.Fatalf("upgrade should have passed validation but didn't: " + err.Error())
	}
}

func Test_Upgrade_Empty_AccountID_Fails(t *testing.T) {
	upgrade, _ := returnValidUpgrade()
	upgrade.AccountID = ""

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}
}

func Test_Upgrade_Invalid_AccountID_Length_Fails(t *testing.T) {
	upgrade, _ := returnValidUpgrade()
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
	upgrade, _ := returnValidUpgrade()
	upgrade.DurationInMonths = 0

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}
}

func Test_Upgrade_StorageLimit_Less_Than_128_Fails(t *testing.T) {
	upgrade, _ := returnValidUpgrade()
	upgrade.NewStorageLimit = 127

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}
}

func Test_Upgrade_No_Eth_Address_Fails(t *testing.T) {
	upgrade, _ := returnValidUpgrade()
	upgrade.EthAddress = ""

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}
}

func Test_Upgrade_Eth_Address_Invalid_Length_Fails(t *testing.T) {
	upgrade, _ := returnValidUpgrade()
	upgrade.EthAddress = utils.RandSeqFromRunes(6, []rune("abcdef01234567890"))

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}
}

func Test_Upgrade_No_Eth_Private_Key_Fails(t *testing.T) {
	upgrade, _ := returnValidUpgrade()
	upgrade.EthPrivateKey = ""

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}
}

func Test_Upgrade_Eth_Private_Key_Invalid_Length_Fails(t *testing.T) {
	upgrade, _ := returnValidUpgrade()
	upgrade.EthPrivateKey = utils.RandSeqFromRunes(6, []rune("abcdef01234567890"))

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}
}

func Test_Upgrade_No_Payment_Status_Fails(t *testing.T) {
	upgrade, _ := returnValidUpgrade()
	upgrade.PaymentStatus = 0

	if err := utils.Validator.Struct(upgrade); err == nil {
		t.Fatalf("upgrade should have failed validation")
	}
}

func Test_Upgrade_GetOrCreateUpgrade(t *testing.T) {
	upgrade, _ := returnValidUpgrade()

	// Test that new upgrade is created
	uPtr, err := GetOrCreateUpgrade(upgrade)
	assert.Nil(t, err)
	assert.Equal(t, uPtr.AccountID, upgrade.AccountID)
	assert.Equal(t, uPtr.EthAddress, upgrade.EthAddress)
	assert.Equal(t, uPtr.NewStorageLimit, upgrade.NewStorageLimit)
	assert.Equal(t, uPtr.OpctCost, upgrade.OpctCost)
	//assert.Equal(t, uPtr.UsdCost, upgrade.UsdCost)

	// simulate generating a new update with the same AccountID and NewStorageLimit
	// although another upgrade already exists--price should not change due to 1 hour price
	// locking
	upgrade2, _ := returnValidUpgrade()
	upgrade2.AccountID = upgrade.AccountID
	upgrade2.NewStorageLimit = upgrade.NewStorageLimit
	upgrade2.OpctCost = 1337.00
	uPtr, err = GetOrCreateUpgrade(upgrade2)
	assert.Nil(t, err)

	// verify AccountID, EthAddress, EthPrivateKey are still the same
	assert.Equal(t, uPtr.AccountID, upgrade.AccountID)
	assert.Equal(t, uPtr.EthAddress, upgrade.EthAddress)
	assert.Equal(t, uPtr.EthPrivateKey, upgrade.EthPrivateKey)

	// verify OpctCost has NOT changed -- price locking keeps the price the same for an hour
	assert.NotEqual(t, upgrade2.OpctCost, uPtr.OpctCost)
	assert.Equal(t, upgrade.OpctCost, uPtr.OpctCost)

	// set the original upgrade's UpgradedAt time to be over an hour old.
	DB.Model(uPtr).UpdateColumn("updated_at", time.Now().Add(-61*time.Minute))
	uPtr, err = GetOrCreateUpgrade(upgrade2)
	assert.Nil(t, err)

	// verify AccountID, EthAddress, EthPrivateKey are still the same
	assert.Equal(t, uPtr.AccountID, upgrade.AccountID)
	assert.Equal(t, uPtr.EthAddress, upgrade.EthAddress)
	assert.Equal(t, uPtr.EthPrivateKey, upgrade.EthPrivateKey)

	// verify OpctCost has changed
	assert.Equal(t, upgrade2.OpctCost, uPtr.OpctCost)
	assert.NotEqual(t, upgrade.OpctCost, uPtr.OpctCost)
}

func Test_Upgrade_GetUpgradeFromAccountIDAndStorageLimits(t *testing.T) {
	upgrade, account := returnValidUpgrade()

	DB.Create(&upgrade)

	upgradeFromDB, err := GetUpgradeFromAccountIDAndStorageLimits(upgrade.AccountID, int(upgrade.NewStorageLimit), int(account.StorageLimit))
	assert.Nil(t, err)
	assert.Equal(t, upgrade.AccountID, upgradeFromDB.AccountID)

	upgradeFromDB, err = GetUpgradeFromAccountIDAndStorageLimits(
		utils.RandSeqFromRunes(AccountIDLength, []rune("abcdef01234567890")), 128, 10)
	assert.NotNil(t, err)
	assert.Equal(t, "", upgradeFromDB.AccountID)
}

func Test_Upgrade_GetUpgradesFromAccountID(t *testing.T) {
	upgrade, _ := returnValidUpgrade()

	DB.Create(&upgrade)

	upgrade2, _ := returnValidUpgrade()
	upgrade2.NewStorageLimit = BasicStorageLimit
	upgrade2.AccountID = upgrade.AccountID

	err := DB.Create(&upgrade2).Error
	assert.Nil(t, err)

	upgrades, err := GetUpgradesFromAccountID(upgrade.AccountID)
	assert.Nil(t, err)

	assert.Equal(t, 2, len(upgrades))
}

func Test_Upgrade_SetUpgradesToNextPaymentStatus(t *testing.T) {
	upgrade, account := returnValidUpgrade()

	DB.Create(&upgrade)

	assert.Equal(t, InitialPaymentInProgress, upgrade.PaymentStatus)

	originalStorageLimit := int(account.StorageLimit)

	SetUpgradesToNextPaymentStatus([]Upgrade{upgrade})
	upgradeFromDB, err := GetUpgradeFromAccountIDAndStorageLimits(
		upgrade.AccountID, int(upgrade.NewStorageLimit), originalStorageLimit)
	assert.Nil(t, err)
	assert.Equal(t, InitialPaymentReceived, upgradeFromDB.PaymentStatus)

	SetUpgradesToNextPaymentStatus([]Upgrade{upgradeFromDB})
	upgradeFromDB, err = GetUpgradeFromAccountIDAndStorageLimits(
		upgrade.AccountID, int(upgrade.NewStorageLimit), originalStorageLimit)
	assert.Nil(t, err)
	assert.Equal(t, GasTransferInProgress, upgradeFromDB.PaymentStatus)

	SetUpgradesToNextPaymentStatus([]Upgrade{upgradeFromDB})
	upgradeFromDB, err = GetUpgradeFromAccountIDAndStorageLimits(
		upgrade.AccountID, int(upgrade.NewStorageLimit), originalStorageLimit)
	assert.Nil(t, err)
	assert.Equal(t, GasTransferComplete, upgradeFromDB.PaymentStatus)

	SetUpgradesToNextPaymentStatus([]Upgrade{upgradeFromDB})
	upgradeFromDB, err = GetUpgradeFromAccountIDAndStorageLimits(
		upgrade.AccountID, int(upgrade.NewStorageLimit), originalStorageLimit)
	assert.Nil(t, err)
	assert.Equal(t, PaymentRetrievalInProgress, upgradeFromDB.PaymentStatus)

	SetUpgradesToNextPaymentStatus([]Upgrade{upgradeFromDB})
	upgradeFromDB, err = GetUpgradeFromAccountIDAndStorageLimits(
		upgrade.AccountID, int(upgrade.NewStorageLimit), originalStorageLimit)
	assert.Nil(t, err)
	assert.Equal(t, PaymentRetrievalComplete, upgradeFromDB.PaymentStatus)

	SetUpgradesToNextPaymentStatus([]Upgrade{upgradeFromDB})
	upgradeFromDB, err = GetUpgradeFromAccountIDAndStorageLimits(
		upgrade.AccountID, int(upgrade.NewStorageLimit), originalStorageLimit)
	assert.Nil(t, err)
	assert.Equal(t, PaymentRetrievalComplete, upgradeFromDB.PaymentStatus)
}

func Test_Upgrade_CheckIfPaid_Has_Paid(t *testing.T) {
	upgrade, _ := returnValidUpgrade()

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return true, utils.TestNetworkID, nil
	}

	if err := DB.Create(&upgrade).Error; err != nil {
		t.Fatalf("should have upgrade account but didn't: " + err.Error())
	}

	paid, _, err := upgrade.CheckIfPaid()
	assert.True(t, paid)
	assert.Nil(t, err)

	upgradesFromDB, _ := GetUpgradesFromAccountID(upgrade.AccountID)

	assert.Equal(t, InitialPaymentReceived, upgradesFromDB[0].PaymentStatus)
}

func Test_Upgrade_CheckIfPaid_Not_Paid(t *testing.T) {
	upgrade, _ := returnValidUpgrade()

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return false, utils.TestNetworkID, nil
	}

	if err := DB.Create(&upgrade).Error; err != nil {
		t.Fatalf("should have upgrade account but didn't: " + err.Error())
	}

	paid, _, err := upgrade.CheckIfPaid()
	assert.False(t, paid)
	assert.Nil(t, err)

	upgradesFromDB, _ := GetUpgradesFromAccountID(upgrade.AccountID)

	assert.Equal(t, InitialPaymentInProgress, upgradesFromDB[0].PaymentStatus)
}

func Test_Upgrade_CheckIfPaid_Error_While_Checking(t *testing.T) {
	upgrade, _ := returnValidUpgrade()

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return false, 0, errors.New("some error")
	}

	if err := DB.Create(&upgrade).Error; err != nil {
		t.Fatalf("should have upgrade account but didn't: " + err.Error())
	}

	paid, _, err := upgrade.CheckIfPaid()
	assert.False(t, paid)
	assert.NotNil(t, err)

	upgradesFromDB, _ := GetUpgradesFromAccountID(upgrade.AccountID)

	assert.Equal(t, InitialPaymentInProgress, upgradesFromDB[0].PaymentStatus)
}

func Test_Upgrade_GetTotalCostInWei(t *testing.T) {
	upgrade, _ := returnValidUpgrade()

	costInWei := upgrade.GetTotalCostInWei()

	assert.Equal(t, "15000000000000000000", costInWei.String())
}

func Test_SetUpgradesToLowerPaymentStatusByUpdateTime(t *testing.T) {
	DeleteUpgradesForTest(t)

	for i := 0; i < 2; i++ {
		upgrade, _ := returnValidUpgrade()
		if err := DB.Create(&upgrade).Error; err != nil {
			t.Fatalf("should have created upgrade but didn't: " + err.Error())
		}
	}

	upgrades := []Upgrade{}
	DB.Find(&upgrades)
	assert.Equal(t, 2, len(upgrades))

	upgrades[0].PaymentStatus = GasTransferInProgress
	upgrades[1].PaymentStatus = GasTransferInProgress

	DB.Save(&upgrades[0])
	DB.Save(&upgrades[1])

	// after cutoff time
	// should NOT get set to lower status
	DB.Exec("UPDATE upgrades set updated_at = ? WHERE account_id = ?;", time.Now().Add(-1*1*24*time.Hour), upgrades[0].AccountID)
	// before cutoff time
	// should get set to lower status
	DB.Exec("UPDATE upgrades set updated_at = ? WHERE account_id = ?;", time.Now().Add(-1*3*24*time.Hour), upgrades[1].AccountID)

	err := SetUpgradesToLowerPaymentStatusByUpdateTime(GasTransferInProgress, time.Now().Add(-1*2*24*time.Hour))
	assert.Nil(t, err)

	upgradesFromDB := []Upgrade{}
	DB.Find(&upgradesFromDB)

	if upgradesFromDB[0].AccountID == upgrades[0].AccountID {
		assert.Equal(t, GasTransferInProgress, upgradesFromDB[0].PaymentStatus)
	}

	if upgradesFromDB[0].AccountID == upgrades[1].AccountID {
		assert.Equal(t, InitialPaymentReceived, upgradesFromDB[0].PaymentStatus)
	}

	if upgradesFromDB[1].AccountID == upgrades[0].AccountID {
		assert.Equal(t, GasTransferInProgress, upgradesFromDB[1].PaymentStatus)
	}

	if upgradesFromDB[1].AccountID == upgrades[1].AccountID {
		assert.Equal(t, InitialPaymentReceived, upgradesFromDB[1].PaymentStatus)
	}
}
