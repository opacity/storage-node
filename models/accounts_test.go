package models

import (
	"errors"
	"testing"

	"encoding/hex"
	"time"

	"math/big"

	"crypto/ecdsa"

	"math/rand"

	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func returnValidAccount() Account {
	ethAddress, privateKey := services.GenerateWallet()
	accountID := utils.RandSeqFromRunes(AccountIDLength, []rune("abcdef01234567890"))

	return Account{
		AccountID:            accountID,
		MonthsInSubscription: DefaultMonthsPerSubscription,
		StorageLocation:      "https://createdInModelsAccountsTest.com/12345",
		StorageLimit:         BasicStorageLimit,
		StorageUsedInByte:    10 * 1e9,
		PaymentStatus:        InitialPaymentInProgress,
		EthAddress:           ethAddress.String(),
		EthPrivateKey:        hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, accountID)),
		ExpiredAt:            time.Now().AddDate(0, DefaultMonthsPerSubscription, 0),
		NetworkIdPaid:        utils.TestNetworkID,
	}
}

func Test_Init_Accounts(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.TestDatabaseURL)
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

func Test_StorageLimit_Less_Than_10_Fails(t *testing.T) {
	account := returnValidAccount()
	account.StorageLimit = 9

	if err := utils.Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_StorageUsedInByte_Less_Than_0_Fails(t *testing.T) {
	account := returnValidAccount()
	account.StorageUsedInByte = int64(-1)

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

	// Add account to DB
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	currentTime := time.Now()
	expirationDate := account.ExpirationDate()

	assert.Equal(t, currentTime.Year()+2, expirationDate.Year())
	assert.Equal(t, currentTime.Month(), expirationDate.Month())
}

func Test_Cost_Returns_Cost(t *testing.T) {
	account := returnValidAccount()
	account.MonthsInSubscription = DefaultMonthsPerSubscription

	cost, err := account.Cost()

	if err != nil {
		t.Fatalf("should have been able to calculate cost")
	}

	assert.Equal(t, BasicSubscriptionDefaultCost, cost)
}

func Test_UpgradeCostInOPCT_Basic_To_Professional_None_Of_Subscription_Has_Passed(t *testing.T) {
	account := returnValidAccount()

	DB.Create(&account)

	upgradeCostInOPCT, err := account.UpgradeCostInOPCT(utils.Env.Plans[1024].StorageInGB, 12)
	assert.Nil(t, err)
	assert.Equal(t, 15.00, math.Ceil(upgradeCostInOPCT))
}

func Test_UpgradeCostInOPCT_None_Of_Subscription_Has_Passed(t *testing.T) {
	account := returnValidAccount()
	account.StorageLimit = StorageLimitType(1024)

	DB.Create(&account)

	upgradeCostInOPCT, err := account.UpgradeCostInOPCT(utils.Env.Plans[2048].StorageInGB, 12)
	assert.Nil(t, err)
	assert.Equal(t, 17.00, math.Ceil(upgradeCostInOPCT))
}

func Test_UpgradeCostInOPCT_Fourth_Of_Subscription_Has_Passed(t *testing.T) {
	account := returnValidAccount()
	account.StorageLimit = StorageLimitType(1024)

	DB.Create(&account)
	timeToSubtract := time.Hour * 24 * (365 / 4)
	account.CreatedAt = time.Now().Add(timeToSubtract * -1)
	DB.Save(&account)

	upgradeCostInOPCT, err := account.UpgradeCostInOPCT(utils.Env.Plans[2048].StorageInGB, 12)
	assert.Nil(t, err)
	assert.Equal(t, 20.00, math.Ceil(upgradeCostInOPCT))
}

func Test_UpgradeCostInOPCT_Half_Of_Subscription_Has_Passed(t *testing.T) {
	account := returnValidAccount()
	account.StorageLimit = StorageLimitType(1024)

	DB.Create(&account)
	timeToSubtract := time.Hour * 24 * (365 / 2)
	account.CreatedAt = time.Now().Add(timeToSubtract * -1)
	DB.Save(&account)

	upgradeCostInOPCT, err := account.UpgradeCostInOPCT(utils.Env.Plans[2048].StorageInGB, 12)
	assert.Nil(t, err)
	assert.Equal(t, 24.00, math.Ceil(upgradeCostInOPCT))
}

func Test_UpgradeCostInOPCT_Three_Fourths_Of_Subscription_Has_Passed(t *testing.T) {
	account := returnValidAccount()
	account.StorageLimit = StorageLimitType(1024)

	DB.Create(&account)
	timeToSubtract := time.Hour * 24 * ((365 / 4) * 3)
	account.CreatedAt = time.Now().Add(timeToSubtract * -1)
	DB.Save(&account)

	upgradeCostInOPCT, err := account.UpgradeCostInOPCT(utils.Env.Plans[2048].StorageInGB, 12)
	assert.Nil(t, err)
	assert.Equal(t, 28.00, math.Ceil(upgradeCostInOPCT))
}

func Test_UpgradeCostInOPCT_Subscription_Expired(t *testing.T) {
	account := returnValidAccount()
	account.StorageLimit = StorageLimitType(1024)

	DB.Create(&account)
	timeToSubtract := time.Hour * 24 * 366
	account.CreatedAt = time.Now().Add(timeToSubtract * -1)
	DB.Save(&account)

	upgradeCostInOPCT, err := account.UpgradeCostInOPCT(utils.Env.Plans[2048].StorageInGB, 12)
	assert.Nil(t, err)
	assert.Equal(t, 32.00, math.Ceil(upgradeCostInOPCT))
}

func Test_UpgradeCostInOPCT_Upgrade_From_Free_Plan_Half_Of_Subscription_Has_Passed(t *testing.T) {
	account := returnValidAccount()
	account.StorageLimit = StorageLimitType(10)

	DB.Create(&account)
	timeToSubtract := time.Hour * 24 * (365 / 2)
	account.CreatedAt = time.Now().Add(timeToSubtract * -1)
	DB.Save(&account)

	upgradeCostInOPCT, err := account.UpgradeCostInOPCT(utils.Env.Plans[2048].StorageInGB, 12)
	assert.Nil(t, err)
	assert.Equal(t, 32.00, math.Ceil(upgradeCostInOPCT))
}

func Test_UpgradeCostInUSD_Half_Of_Subscription_Has_Passed(t *testing.T) {
	t.Skip("will be added back once prices are set")

	account := returnValidAccount()
	account.StorageLimit = StorageLimitType(1024)

	DB.Create(&account)
	timeToSubtract := time.Hour * 24 * (365 / 2)
	account.CreatedAt = time.Now().Add(timeToSubtract * -1)
	DB.Save(&account)

	upgradeCostInUSD, err := account.UpgradeCostInUSD(utils.Env.Plans[2048].StorageInGB, 12)
	assert.Nil(t, err)
	assert.Equal(t, 100.00, math.Ceil(upgradeCostInUSD))
}

func Test_UpgradeCostInUSD_Fourth_Of_Subscription_Has_Passed(t *testing.T) {
	t.Skip("will be added back once prices are set")

	account := returnValidAccount()
	account.StorageLimit = StorageLimitType(1024)

	DB.Create(&account)
	timeToSubtract := time.Hour * 24 * (365 / 4)
	account.CreatedAt = time.Now().Add(timeToSubtract * -1)
	DB.Save(&account)

	upgradeCostInUSD, err := account.UpgradeCostInUSD(utils.Env.Plans[2048].StorageInGB, 12)
	assert.Nil(t, err)
	assert.Equal(t, 75.00, math.Ceil(upgradeCostInUSD))
}

func Test_UpgradeCostInUSD_Three_Fourths_Of_Subscription_Has_Passed(t *testing.T) {
	t.Skip("will be added back once prices are set")

	account := returnValidAccount()
	account.StorageLimit = StorageLimitType(1024)

	DB.Create(&account)
	timeToSubtract := time.Hour * 24 * ((365 / 4) * 3)
	account.CreatedAt = time.Now().Add(timeToSubtract * -1)
	DB.Save(&account)

	upgradeCostInUSD, err := account.UpgradeCostInUSD(utils.Env.Plans[2048].StorageInGB, 12)
	assert.Nil(t, err)
	assert.Equal(t, 125.00, math.Ceil(upgradeCostInUSD))
}

func Test_UpgradeCostInUSD_Subscription_Expired(t *testing.T) {
	t.Skip("will be added back once prices are set")

	account := returnValidAccount()
	account.StorageLimit = StorageLimitType(1024)

	DB.Create(&account)
	timeToSubtract := time.Hour * 24 * 366
	account.CreatedAt = time.Now().Add(timeToSubtract * -1)
	DB.Save(&account)

	upgradeCostInUSD, err := account.UpgradeCostInUSD(utils.Env.Plans[2048].StorageInGB, 12)
	assert.Nil(t, err)
	assert.Equal(t, 150.00, math.Ceil(upgradeCostInUSD))
}

func Test_UpgradeCostInUSD_Upgrade_From_Free_Plan_Half_Of_Subscription_Has_Passed(t *testing.T) {
	t.Skip("will be added back once prices are set")

	account := returnValidAccount()
	account.StorageLimit = StorageLimitType(10)

	DB.Create(&account)
	timeToSubtract := time.Hour * 24 * (365 / 2)
	account.CreatedAt = time.Now().Add(timeToSubtract * -1)
	DB.Save(&account)

	upgradeCostInUSD, err := account.UpgradeCostInUSD(utils.Env.Plans[2048].StorageInGB, 12)
	assert.Nil(t, err)
	assert.Equal(t, 150.00, math.Ceil(upgradeCostInUSD))
}

func Test_GetTotalCostInWei(t *testing.T) {
	account := returnValidAccount()
	account.MonthsInSubscription = DefaultMonthsPerSubscription

	costInWei := account.GetTotalCostInWei()

	assert.Equal(t, big.NewInt(2000000000000000000).String(), costInWei.String())
}

func Test_CheckIfPaid_Has_Paid(t *testing.T) {
	account := returnValidAccount()
	account.MonthsInSubscription = DefaultMonthsPerSubscription

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return true, utils.TestNetworkID, nil
	}

	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	paid, _, err := account.CheckIfPaid()
	assert.True(t, paid)
	assert.Nil(t, err)

	accountFromDB, _ := GetAccountById(account.AccountID)

	assert.Equal(t, InitialPaymentReceived, accountFromDB.PaymentStatus)
}

func Test_CheckIfPaid_Not_Paid(t *testing.T) {
	account := returnValidAccount()
	account.MonthsInSubscription = DefaultMonthsPerSubscription

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return false, utils.TestNetworkID, nil
	}

	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	paid, _, err := account.CheckIfPaid()
	assert.False(t, paid)
	assert.Nil(t, err)

	accountFromDB, _ := GetAccountById(account.AccountID)

	assert.Equal(t, InitialPaymentInProgress, accountFromDB.PaymentStatus)
}

func Test_CheckIfPaid_Error_While_Checking(t *testing.T) {
	account := returnValidAccount()
	account.MonthsInSubscription = DefaultMonthsPerSubscription

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return false, 0, errors.New("some error")
	}

	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	paid, _, err := account.CheckIfPaid()
	assert.False(t, paid)
	assert.NotNil(t, err)

	accountFromDB, _ := GetAccountById(account.AccountID)

	assert.Equal(t, InitialPaymentInProgress, accountFromDB.PaymentStatus)
}

func Test_CheckIfPending_Is_Pending(t *testing.T) {
	account := returnValidAccount()

	BackendManager.CheckIfPending = func(address common.Address) bool {
		return true
	}

	pending := account.CheckIfPending()
	assert.True(t, pending)
}

func Test_CheckIfPending_Is_Not_Pending(t *testing.T) {
	account := returnValidAccount()

	BackendManager.CheckIfPending = func(address common.Address) bool {
		return false
	}

	pending := account.CheckIfPending()
	assert.False(t, pending)
}

func Test_CheckIfPending_Error_While_Checking(t *testing.T) {
	account := returnValidAccount()

	BackendManager.CheckIfPending = func(address common.Address) bool {
		return false
	}

	pending := account.CheckIfPending()
	assert.False(t, pending)
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
	account.ExpiredAt = savedAccount.ExpiredAt
	assert.Equal(t, account, savedAccount)
}

func Test_HasEnoughSpaceToUploadFile(t *testing.T) {
	account := returnValidAccount()
	account.PaymentStatus = PaymentRetrievalComplete
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	startingSpaceUsed := account.StorageUsedInByte
	assert.Nil(t, account.UseStorageSpaceInByte(10*1e9 /* Upload 10GB. */))
	accountFromDB, _ := GetAccountById(account.AccountID)
	assert.True(t, startingSpaceUsed == accountFromDB.StorageUsedInByte-10*1e9)
}

func Test_NoEnoughSpaceToUploadFile(t *testing.T) {
	account := returnValidAccount()
	account.PaymentStatus = PaymentRetrievalComplete
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	assert.NotNil(t, account.UseStorageSpaceInByte(123*1e9 /* Upload 95GB. */))
}

func Test_DeductSpaceUsed(t *testing.T) {
	account := returnValidAccount()
	account.PaymentStatus = PaymentRetrievalComplete
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	startingSpaceUsed := account.StorageUsedInByte
	assert.Nil(t, account.UseStorageSpaceInByte(-9*1e9 /* Deleted 9 GB file. */))
	accountFromDB, _ := GetAccountById(account.AccountID)
	assert.True(t, startingSpaceUsed == accountFromDB.StorageUsedInByte+9*1e9)
}

func Test_DeductSpaceUsed_Too_Much_Deducted(t *testing.T) {
	account := returnValidAccount()
	account.PaymentStatus = PaymentRetrievalComplete
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	assert.NotNil(t, account.UseStorageSpaceInByte(-11*1e9 /* Deduct 11 GB file but only 10 GB uploaded. */))
	accountFromDB, _ := GetAccountById(account.AccountID)
	assert.True(t, accountFromDB.StorageUsedInByte == account.StorageUsedInByte)
}

func Test_Space_Updates_at_Scale(t *testing.T) {
	account := returnValidAccount()
	account.StorageUsedInByte = 0
	account.PaymentStatus = InitialPaymentReceived
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	numIntendedUpdates := 100
	numAdds := 0
	numDeletes := 0
	byteValues := make(map[int]int64)

	for i := 0; i < numIntendedUpdates; i++ {
		byteValues[i] = rand.Int63n(10000)
	}

	for i := 0; i < numIntendedUpdates; i++ {
		go func(byteValue int64, account Account) {
			assert.Nil(t, account.UseStorageSpaceInByte(byteValue))
			numAdds++
		}(byteValues[i], account)
	}

	for {
		if numAdds == numIntendedUpdates {
			break
		}
	}

	accountFromDB, _ := GetAccountById(account.AccountID)
	assert.NotEqual(t, int64(0), accountFromDB.StorageUsedInByte)

	for i := 0; i < numIntendedUpdates; i++ {
		go func(byteValue int64, account Account) {
			assert.Nil(t, account.UseStorageSpaceInByte(-1*byteValue))
			numDeletes++
		}(byteValues[i], account)
	}

	for {
		if numDeletes == numIntendedUpdates {
			break
		}
	}

	accountFromDB, _ = GetAccountById(account.AccountID)
	assert.Equal(t, int64(0), accountFromDB.StorageUsedInByte)
}

func Test_MaxAllowedMetadataSizeInBytes(t *testing.T) {
	// This test relies upon TestFileStoragePerMetadataInMB
	// and TestMaxPerMetadataSizeInMB defined in utils/env.go.
	// If those values are changed this test will fail.
	expectedMaxAllowedMetadataSizeInBytes := int64(200 * 1e6)

	account := returnValidAccount()
	account.StorageLimit = BasicStorageLimit
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	actualMaxAllowedMetadataSizeInBytes := account.MaxAllowedMetadataSizeInBytes()

	assert.Equal(t, expectedMaxAllowedMetadataSizeInBytes, actualMaxAllowedMetadataSizeInBytes)
}

func Test_MaxAllowedMetadatas(t *testing.T) {
	// This test relies upon TestFileStoragePerMetadataInMB
	// and TestMaxPerMetadataSizeInMB defined in utils/env.go.
	// If those values are changed this test will fail.
	expectedMaxAllowedMetadatas := utils.Env.Plans[int(BasicStorageLimit)].MaxFolders

	account := returnValidAccount()
	account.StorageLimit = BasicStorageLimit
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	actualMaxAllowedMetadatas := account.MaxAllowedMetadatas()

	assert.Equal(t, expectedMaxAllowedMetadatas, actualMaxAllowedMetadatas)
}

func Test_CanAddNewMetadata(t *testing.T) {
	// This test relies upon TestFileStoragePerMetadataInMB
	// and TestMaxPerMetadataSizeInMB defined in utils/env.go.
	// If those values are changed this test will fail.
	account := returnValidAccount()
	account.TotalFolders = 1998
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	assert.True(t, account.CanAddNewMetadata())
	account.TotalFolders++
	assert.True(t, account.CanAddNewMetadata())
	account.TotalFolders++
	assert.False(t, account.CanAddNewMetadata())
}

func Test_CanRemoveMetadata(t *testing.T) {
	// This test relies upon TestFileStoragePerMetadataInMB
	// and TestMaxPerMetadataSizeInMB defined in utils/env.go.
	// If those values are changed this test will fail.

	account := returnValidAccount()
	account.TotalFolders = 1
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	assert.True(t, account.CanRemoveMetadata())
	account.TotalFolders = 0
	assert.False(t, account.CanRemoveMetadata())
}

func Test_CanUpdateMetadata(t *testing.T) {
	// This test relies upon TestFileStoragePerMetadataInMB
	// and TestMaxPerMetadataSizeInMB defined in utils/env.go.
	// If those values are changed this test will fail.

	account := returnValidAccount()
	account.TotalMetadataSizeInBytes = int64(100 * 1e6)
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	assert.True(t, account.CanUpdateMetadata(100, 50e6))
	account.TotalMetadataSizeInBytes = int64(200 * 1e6)
	assert.False(t, account.CanUpdateMetadata(3.2e10, 50e6))
}

func Test_IncrementMetadataCount(t *testing.T) {
	// This test relies upon TestFileStoragePerMetadataInMB
	// and TestMaxPerMetadataSizeInMB defined in utils/env.go.
	// If those values are changed this test will fail.

	account := returnValidAccount()
	account.TotalFolders = utils.Env.Plans[int(BasicStorageLimit)].MaxFolders - 2
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	err := account.IncrementMetadataCount()
	assert.Nil(t, err)

	accountFromDB, _ := GetAccountById(account.AccountID)
	assert.True(t, accountFromDB.TotalFolders == utils.Env.Plans[int(BasicStorageLimit)].MaxFolders-1)

	err = account.IncrementMetadataCount()
	assert.Nil(t, err)

	accountFromDB, _ = GetAccountById(account.AccountID)
	assert.True(t, accountFromDB.TotalFolders == utils.Env.Plans[int(BasicStorageLimit)].MaxFolders)

	err = account.IncrementMetadataCount()
	assert.NotNil(t, err)

	accountFromDB, _ = GetAccountById(account.AccountID)
	assert.True(t, accountFromDB.TotalFolders == utils.Env.Plans[int(BasicStorageLimit)].MaxFolders)
}

func Test_DecrementMetadataCount(t *testing.T) {
	// This test relies upon TestFileStoragePerMetadataInMB
	// and TestMaxPerMetadataSizeInMB defined in utils/env.go.
	// If those values are changed this test will fail.

	account := returnValidAccount()
	account.TotalFolders = 1
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	err := account.DecrementMetadataCount()
	assert.Nil(t, err)

	accountFromDB, _ := GetAccountById(account.AccountID)
	assert.True(t, accountFromDB.TotalFolders == 0)

	err = account.DecrementMetadataCount()
	assert.NotNil(t, err)
}

func Test_UpdateMetadataSizeInBytes(t *testing.T) {
	// This test relies upon TestFileStoragePerMetadataInMB
	// and TestMaxPerMetadataSizeInMB defined in utils/env.go.
	// If those values are changed this test will fail.

	account := returnValidAccount()
	account.TotalMetadataSizeInBytes = 100
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	assert.Nil(t, account.UpdateMetadataSizeInBytes(100, 3.2e6))
	account.TotalMetadataSizeInBytes = 200e6
	assert.NotNil(t, account.UpdateMetadataSizeInBytes(200e6, 300e6))
}

func Test_RemoveMetadata(t *testing.T) {
	// This test relies upon TestFileStoragePerMetadataInMB
	// and TestMaxPerMetadataSizeInMB defined in utils/env.go.
	// If those values are changed this test will fail.

	account := returnValidAccount()
	account.TotalMetadataSizeInBytes = 100
	account.TotalFolders = 1
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	assert.Nil(t, account.RemoveMetadata(100))

	accountFromDB, _ := GetAccountById(account.AccountID)
	assert.True(t, accountFromDB.TotalFolders == 0)
	assert.True(t, accountFromDB.TotalMetadataSizeInBytes == int64(0))

	account.TotalMetadataSizeInBytes = 100
	account.TotalFolders = 1
	DB.Save(&account)

	assert.NotNil(t, account.RemoveMetadata(101))
	account.TotalMetadataSizeInBytes = 100
	account.TotalFolders = 0
	DB.Save(&account)

	assert.NotNil(t, account.RemoveMetadata(100))
}

func Test_UpgradeAccount(t *testing.T) {
	account := returnValidAccount()

	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}
	account.CreatedAt = time.Now().Add(time.Hour * 24 * (365 / 2) * -1)
	DB.Save(&account)

	startingExpirationDate := account.ExpirationDate()
	startingStorageLimit := account.StorageLimit
	startingMonthsInSubscription := account.MonthsInSubscription

	err := account.UpgradeAccount(1024, 12)

	assert.Nil(t, err)
	assert.Equal(t, 1024, int(account.StorageLimit))

	newExpirationDate := account.ExpirationDate()
	assert.NotEqual(t, startingExpirationDate, newExpirationDate)
	assert.NotEqual(t, startingMonthsInSubscription, account.MonthsInSubscription)
	assert.NotEqual(t, startingStorageLimit, int(account.MonthsInSubscription))
}

func Test_UpgradeAccount_Invalid_Storage_Value(t *testing.T) {
	account := returnValidAccount()

	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}
	account.CreatedAt = time.Now().Add(time.Hour * 24 * (365 / 2) * -1)
	DB.Save(&account)

	startingExpirationDate := account.ExpirationDate()
	startingStorageLimit := account.StorageLimit
	startingMonthsInSubscription := account.MonthsInSubscription

	err := account.UpgradeAccount(1023, 12)

	assert.NotNil(t, err)
	assert.Equal(t, startingStorageLimit, account.StorageLimit)

	newExpirationDate := account.ExpirationDate()
	assert.Equal(t, startingExpirationDate, newExpirationDate)
	assert.Equal(t, startingMonthsInSubscription, account.MonthsInSubscription)
}

func Test_RenewAccount(t *testing.T) {
	account := returnValidAccount()
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	originalMonthsInSubscription := account.MonthsInSubscription

	err := account.RenewAccount()
	assert.Nil(t, err)

	accountFromDB, _ := GetAccountById(account.AccountID)

	assert.Equal(t, originalMonthsInSubscription+12, accountFromDB.MonthsInSubscription)
}

func Test_GetAccountById(t *testing.T) {
	account := returnValidAccount()
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	accountFromDB, _ := GetAccountById(account.AccountID)
	assert.Equal(t, accountFromDB.AccountID, accountFromDB.AccountID)
	assert.NotEqual(t, "", accountFromDB.AccountID)

	account = returnValidAccount()

	accountFromDB, err := GetAccountById(account.AccountID)
	assert.NotNil(t, err)
	assert.Equal(t, "", accountFromDB.AccountID)
}

func Test_CreateSpaceUsedReport(t *testing.T) {
	expectedSpaceAllotted := int(4 * BasicStorageLimit)
	expectedSpaceUsed := 234.56 * 1e9

	DeleteAccountsForTest(t)

	for i := 0; i < 4; i++ {
		accountPaid := returnValidAccount()
		accountPaid.StorageUsedInByte = int64(expectedSpaceUsed / 4)
		accountPaid.PaymentStatus = PaymentStatusType(utils.RandIndex(5) + 2)
		if err := DB.Create(&accountPaid).Error; err != nil {
			t.Fatalf("should have created account but didn't: " + err.Error())
		}
	}

	for i := 0; i < 4; i++ {
		accountUnpaid := returnValidAccount()
		accountUnpaid.StorageUsedInByte = int64(expectedSpaceUsed / 4)
		if err := DB.Create(&accountUnpaid).Error; err != nil {
			t.Fatalf("should have created account but didn't: " + err.Error())
		}
	}

	spaceReport := CreateSpaceUsedReport()

	assert.Equal(t, expectedSpaceAllotted, spaceReport.SpaceAllottedSum)
	assert.Equal(t, expectedSpaceUsed, spaceReport.SpaceUsedSum)
}

func Test_CreateSpaceUsedReportForPlanType(t *testing.T) {
	expectedSpaceAllottedBasic := int(4 * BasicStorageLimit)
	expectedSpaceAllottedProfessional := int(4 * ProfessionalStorageLimit)
	expectedSpaceUsed := 234.56 * 1e9

	DeleteAccountsForTest(t)

	for i := 0; i < 4; i++ {
		accountPaid := returnValidAccount()
		accountPaid.StorageUsedInByte = int64(expectedSpaceUsed / 4)
		accountPaid.PaymentStatus = PaymentStatusType(utils.RandIndex(5) + 2)
		if err := DB.Create(&accountPaid).Error; err != nil {
			t.Fatalf("should have created account but didn't: " + err.Error())
		}
	}
	for i := 0; i < 4; i++ {
		accountPaid := returnValidAccount()
		accountPaid.StorageUsedInByte = int64(expectedSpaceUsed / 4)
		accountPaid.StorageLimit = ProfessionalStorageLimit
		accountPaid.PaymentStatus = PaymentStatusType(utils.RandIndex(5) + 2)
		if err := DB.Create(&accountPaid).Error; err != nil {
			t.Fatalf("should have created account but didn't: " + err.Error())
		}
	}

	for i := 0; i < 4; i++ {
		accountUnpaid := returnValidAccount()
		accountUnpaid.StorageUsedInByte = int64(expectedSpaceUsed / 4)
		if err := DB.Create(&accountUnpaid).Error; err != nil {
			t.Fatalf("should have created account but didn't: " + err.Error())
		}
	}
	for i := 0; i < 4; i++ {
		accountUnpaid := returnValidAccount()
		accountUnpaid.StorageUsedInByte = int64(expectedSpaceUsed / 4)
		accountUnpaid.StorageLimit = ProfessionalStorageLimit
		if err := DB.Create(&accountUnpaid).Error; err != nil {
			t.Fatalf("should have created account but didn't: " + err.Error())
		}
	}

	spaceReport := CreateSpaceUsedReportForPlanType(BasicStorageLimit)

	assert.Equal(t, expectedSpaceAllottedBasic, spaceReport.SpaceAllottedSum)
	assert.Equal(t, expectedSpaceUsed, spaceReport.SpaceUsedSum)

	spaceReport = CreateSpaceUsedReportForPlanType(ProfessionalStorageLimit)

	assert.Equal(t, expectedSpaceAllottedProfessional, spaceReport.SpaceAllottedSum)
	assert.Equal(t, expectedSpaceUsed, spaceReport.SpaceUsedSum)
}

func Test_CalculatePercentSpaceUsed(t *testing.T) {
	expectedPercent := 50.0

	spaceReport := SpaceReport{
		SpaceAllottedSum: 100,
		SpaceUsedSum:     float64(expectedPercent * 1e9),
	}

	percentUsed := CalculatePercentSpaceUsed(spaceReport)

	assert.Equal(t, expectedPercent, percentUsed)
}

func Test_PurgeOldUnpaidAccounts(t *testing.T) {
	DeleteAccountsForTest(t)

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
	accounts[0].CreatedAt = time.Now().Add(-1 * time.Duration(utils.Env.AccountRetentionDays-1) * 24 * time.Hour)
	accounts[0].PaymentStatus = InitialPaymentReceived

	// before cutoff time but payment has been received
	// should NOT get purged
	accounts[1].CreatedAt = time.Now().Add(-1 * time.Duration(utils.Env.AccountRetentionDays+1) * 24 * time.Hour)
	accounts[1].PaymentStatus = InitialPaymentReceived

	// after cutoff time, payment still in progress
	// should NOT get purged
	accounts[2].CreatedAt = time.Now().Add(-1 * time.Duration(utils.Env.AccountRetentionDays-1) * 24 * time.Hour)
	accounts[2].PaymentStatus = InitialPaymentInProgress

	// before cutoff time, payment still in progress
	// this one should get purged
	accounts[3].CreatedAt = time.Now().Add(-1 * time.Duration(utils.Env.AccountRetentionDays+1) * 24 * time.Hour)
	accounts[3].StorageUsedInByte = 0
	accounts[3].PaymentStatus = InitialPaymentInProgress

	accountToBeDeletedID := accounts[3].AccountID

	DB.Save(&accounts[0])
	DB.Save(&accounts[1])
	DB.Save(&accounts[2])
	DB.Save(&accounts[3])

	PurgeOldUnpaidAccounts(utils.Env.AccountRetentionDays)

	accounts = []Account{}
	DB.Find(&accounts)
	assert.Equal(t, 3, len(accounts))

	accounts = []Account{}
	DB.Where("account_id = ?", accountToBeDeletedID).Find(&accounts)
	assert.Equal(t, 0, len(accounts))
}

func Test_PurgeOldUnpaidAccounts_Stripe_Payment_Is_Deleted(t *testing.T) {
	DeleteAccountsForTest(t)
	DeleteStripePaymentsForTest(t)
	stripePayment, _ := returnValidStripePaymentForTest()
	DB.Create(&stripePayment)

	account, _ := GetAccountById(stripePayment.AccountID)
	account.CreatedAt = time.Now().Add(time.Duration(utils.Env.AccountRetentionDays+1) * -24 * time.Hour)
	account.StorageUsedInByte = 0
	account.PaymentStatus = InitialPaymentInProgress
	DB.Save(&account)

	accounts := []Account{}
	DB.Find(&accounts)
	assert.Equal(t, 1, len(accounts))

	stripePayments := []StripePayment{}
	DB.Find(&stripePayments)
	assert.Equal(t, 1, len(stripePayments))

	PurgeOldUnpaidAccounts(utils.Env.AccountRetentionDays)

	accounts = []Account{}
	DB.Find(&accounts)
	assert.Equal(t, 0, len(accounts))

	stripePayments = []StripePayment{}
	DB.Find(&stripePayments)
	assert.Equal(t, 0, len(stripePayments))
}

func Test_GetAccountsByPaymentStatus(t *testing.T) {
	DeleteAccountsForTest(t)
	// for each payment status, check that we can get the accounts of that status and that the account IDs
	// of the accounts returned from GetAccountsByPaymentStatus match the accounts we created for the test
	for paymentStatus := InitialPaymentInProgress; paymentStatus <= PaymentRetrievalComplete; paymentStatus++ {
		if err := DB.Where("payment_status = ?", paymentStatus).Unscoped().Delete(&Account{}).Error; err != nil {
			t.Fatalf("should have deleted accounts but didn't: " + err.Error())
		}
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

func Test_CountAccountsByPaymentStatus(t *testing.T) {
	DeleteAccountsForTest(t)
	// for each payment status, check that we can get the accounts of that status and that the account IDs
	// of the accounts returned from GetAccountsByPaymentStatus match the accounts we created for the test
	for paymentStatus := InitialPaymentInProgress; paymentStatus <= PaymentRetrievalComplete; paymentStatus++ {
		if err := DB.Where("payment_status = ?", paymentStatus).Unscoped().Delete(&Account{}).Error; err != nil {
			t.Fatalf("should have deleted accounts but didn't: " + err.Error())
		}
		account := returnValidAccount()
		account.PaymentStatus = paymentStatus
		if err := DB.Create(&account).Error; err != nil {
			t.Fatalf("should have created account but didn't: " + err.Error())
		}
		count, err := CountAccountsByPaymentStatus(paymentStatus)
		assert.Nil(t, err)
		assert.Equal(t, 1, count)
	}
}

func Test_CountPaidAccountsByPlanType(t *testing.T) {
	DeleteAccountsForTest(t)
	// for each payment status, check that we can get the accounts of that status and that the account IDs
	// of the accounts returned from GetAccountsByPaymentStatus match the accounts we created for the test
	for i := 0; i < 4; i++ {
		account := returnValidAccount()
		account.PaymentStatus = PaymentStatusType(utils.RandIndex(5) + 2)
		if err := DB.Create(&account).Error; err != nil {
			t.Fatalf("should have created account but didn't: " + err.Error())
		}
	}

	for i := 0; i < 4; i++ {
		account := returnValidAccount()
		account.StorageLimit = ProfessionalStorageLimit
		account.PaymentStatus = PaymentStatusType(utils.RandIndex(5) + 2)
		if err := DB.Create(&account).Error; err != nil {
			t.Fatalf("should have created account but didn't: " + err.Error())
		}
	}

	count, _ := CountPaidAccountsByPlanType(BasicStorageLimit)
	assert.Equal(t, 4, count)

	count, _ = CountPaidAccountsByPlanType(ProfessionalStorageLimit)
	assert.Equal(t, 4, count)
}

func Test_SetAccountsToNextPaymentStatus(t *testing.T) {

	for paymentStatus := InitialPaymentInProgress; paymentStatus <= PaymentRetrievalComplete; paymentStatus++ {
		DeleteAccountsForTest(t)

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
	DeleteAccountsForTest(t)

	account := returnValidAccount()
	account.PaymentStatus = InitialPaymentInProgress

	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return true, utils.TestNetworkID, nil
	}

	// grab the account from the DB
	accountFromDB, _ := GetAccountById(account.AccountID)

	verifyPaymentStatusExpectations(t, accountFromDB, InitialPaymentInProgress, InitialPaymentReceived, handleAccountWithPaymentInProgress)
}

func Test_handleAccountWithPaymentInProgress_has_not_paid(t *testing.T) {
	DeleteAccountsForTest(t)
	account := returnValidAccount()
	account.PaymentStatus = InitialPaymentInProgress

	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return false, utils.TestNetworkID, nil
	}

	// grab the account from the DB
	accountFromDB, _ := GetAccountById(account.AccountID)

	verifyPaymentStatusExpectations(t, accountFromDB, InitialPaymentInProgress, InitialPaymentInProgress, handleAccountWithPaymentInProgress)
}

func Test_handleAccountThatNeedsGas_transfer_success(t *testing.T) {
	DeleteAccountsForTest(t)
	account := returnValidAccount()
	account.PaymentStatus = InitialPaymentReceived
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	services.EthOpsWrapper.TransferETH = func(ethWrapper *services.Eth, fromAddress common.Address, fromPrivKey *ecdsa.PrivateKey,
		toAddr common.Address, amount *big.Int) (types.Transactions, string, int64, error) {
		// not returning anything important for the first three return values because
		// handleAccountThatNeedsGas only cares about the 4th return value which will
		// be an error or nil
		return types.Transactions{}, "", -1, nil
	}

	verifyPaymentStatusExpectations(t, account, InitialPaymentReceived, GasTransferInProgress, handleAccountThatNeedsGas)
}

func Test_handleAccountThatNeedsGas_transfer_error(t *testing.T) {
	DeleteAccountsForTest(t)
	account := returnValidAccount()
	account.PaymentStatus = InitialPaymentReceived
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	services.EthOpsWrapper.TransferETH = func(ethWrapper *services.Eth, fromAddress common.Address, fromPrivKey *ecdsa.PrivateKey,
		toAddr common.Address, amount *big.Int) (types.Transactions, string, int64, error) {
		// not returning anything important for the first three return values because
		// handleAccountThatNeedsGas only cares about the 4th return value which will
		// be an error or nil
		return types.Transactions{}, "", -1, errors.New("SOMETHING HAPPENED")
	}

	verifyPaymentStatusExpectations(t, account, InitialPaymentReceived, InitialPaymentReceived, handleAccountThatNeedsGas)

}

func Test_handleAccountReceivingGas_gas_received(t *testing.T) {
	DeleteAccountsForTest(t)
	account := returnValidAccount()
	account.PaymentStatus = GasTransferInProgress
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	services.EthOpsWrapper.GetETHBalance = func(ethWrapper *services.Eth, addr common.Address) *big.Int {
		return big.NewInt(1)
	}

	verifyPaymentStatusExpectations(t, account, GasTransferInProgress, GasTransferComplete, handleAccountReceivingGas)

}

func Test_handleAccountReceivingGas_gas_not_received(t *testing.T) {
	DeleteAccountsForTest(t)
	account := returnValidAccount()
	account.PaymentStatus = GasTransferInProgress
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	services.EthOpsWrapper.GetETHBalance = func(ethWrapper *services.Eth, addr common.Address) *big.Int {
		return big.NewInt(-1)
	}

	verifyPaymentStatusExpectations(t, account, GasTransferInProgress, GasTransferInProgress, handleAccountReceivingGas)

}

func Test_handleAccountReadyForCollection_transfer_success(t *testing.T) {
	DeleteAccountsForTest(t)
	account := returnValidAccount()
	account.PaymentStatus = GasTransferComplete
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	services.EthOpsWrapper.GetETHBalance = func(ethWrapper *services.Eth, addr common.Address) *big.Int {
		return big.NewInt(1)
	}
	services.EthOpsWrapper.GetTokenBalance = func(ethWrapper *services.Eth, addr common.Address) *big.Int {
		return big.NewInt(1)
	}
	services.EthOpsWrapper.TransferToken = func(ethWrapper *services.Eth, from common.Address, privateKey *ecdsa.PrivateKey, to common.Address,
		opctAmount big.Int, gasPrice *big.Int) (bool, string, int64) {
		// all that handleAccountReadyForCollection cares about is the first return value
		return true, "", 1
	}

	verifyPaymentStatusExpectations(t, account, GasTransferComplete, PaymentRetrievalInProgress, handleAccountReadyForCollection)
}

func Test_handleAccountReadyForCollection_transfer_failed(t *testing.T) {
	DeleteAccountsForTest(t)
	account := returnValidAccount()
	account.PaymentStatus = GasTransferComplete
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	services.EthOpsWrapper.GetETHBalance = func(ethWrapper *services.Eth, addr common.Address) *big.Int {
		return big.NewInt(1)
	}
	services.EthOpsWrapper.GetTokenBalance = func(ethWrapper *services.Eth, addr common.Address) *big.Int {
		return big.NewInt(1)
	}
	services.EthOpsWrapper.TransferToken = func(ethWrapper *services.Eth, from common.Address, privateKey *ecdsa.PrivateKey, to common.Address,
		opctAmount big.Int, gasPrice *big.Int) (bool, string, int64) {
		// all that handleAccountReadyForCollection cares about is the first return value
		return false, "", 1
	}

	verifyPaymentStatusExpectations(t, account, GasTransferComplete, GasTransferComplete, handleAccountReadyForCollection)

}

func Test_handleAccountWithCollectionInProgress_balance_not_found(t *testing.T) {
	DeleteAccountsForTest(t)
	account := returnValidAccount()
	account.PaymentStatus = PaymentRetrievalInProgress
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	services.EthOpsWrapper.GetTokenBalance = func(ethWrapper *services.Eth, addr common.Address) *big.Int {
		return big.NewInt(1)
	}

	verifyPaymentStatusExpectations(t, account, PaymentRetrievalInProgress, PaymentRetrievalInProgress, handleAccountWithCollectionInProgress)
}

func Test_handleAccountWithCollectionInProgress_balance_found(t *testing.T) {
	DeleteAccountsForTest(t)
	account := returnValidAccount()
	account.PaymentStatus = PaymentRetrievalInProgress
	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	services.EthOpsWrapper.GetTokenBalance = func(ethWrapper *services.Eth, addr common.Address) *big.Int {
		return big.NewInt(0)
	}

	verifyPaymentStatusExpectations(t, account, PaymentRetrievalInProgress, PaymentRetrievalComplete, handleAccountWithCollectionInProgress)
}

func Test_GetAllExpiredAccounts(t *testing.T) {
	DeleteAccountsForTest(t)
	accountExpired := returnValidAccount()
	if err := DB.Create(&accountExpired).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}
	DB.Model(&accountExpired).UpdateColumn("expired_at", time.Now().Add(-1*time.Hour*24))

	accountNotExpired := returnValidAccount()
	if err := DB.Create(&accountNotExpired).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}
	DB.Model(&accountNotExpired).UpdateColumn("expired_at", time.Now().Add(1*time.Hour*24))

	accounts, err := GetAllExpiredAccounts(time.Now())

	assert.Nil(t, err)

	assert.Equal(t, 1, len(accounts))
	assert.Equal(t, accountExpired.AccountID, accounts[0].AccountID)
}

func Test_DeleteExpiredAccounts(t *testing.T) {
	DeleteAccountsForTest(t)
	DeleteExpiredAccountsForTest(t)
	accountExpired := returnValidAccount()
	if err := DB.Create(&accountExpired).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}
	DB.Model(&accountExpired).UpdateColumn("expired_at", time.Now().Add(-1*time.Hour*24))

	accountNotExpired := returnValidAccount()
	if err := DB.Create(&accountNotExpired).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}
	DB.Model(&accountNotExpired).UpdateColumn("expired_at", time.Now().Add(1*time.Hour*24))

	err := DeleteExpiredAccounts(time.Now())
	assert.Nil(t, err)

	accounts := []Account{}
	DB.Find(&accounts)
	assert.Equal(t, 1, len(accounts))
	assert.Equal(t, accountNotExpired.AccountID, accounts[0].AccountID)

	expiredAccounts := []ExpiredAccount{}
	DB.Find(&expiredAccounts)
	assert.Equal(t, 1, len(expiredAccounts))
	assert.Equal(t, accountExpired.AccountID, expiredAccounts[0].AccountID)
}

func Test_SetAccountsToLowerPaymentStatusByUpdateTime(t *testing.T) {
	DeleteAccountsForTest(t)

	for i := 0; i < 2; i++ {
		account := returnValidAccount()
		if err := DB.Create(&account).Error; err != nil {
			t.Fatalf("should have created account but didn't: " + err.Error())
		}
	}

	accounts := []Account{}
	DB.Find(&accounts)
	assert.Equal(t, 2, len(accounts))

	accounts[0].PaymentStatus = GasTransferInProgress
	accounts[1].PaymentStatus = GasTransferInProgress

	DB.Save(&accounts[0])
	DB.Save(&accounts[1])

	// after cutoff time
	// should NOT get set to lower status
	DB.Exec("UPDATE accounts set updated_at = ? WHERE account_id = ?;", time.Now().Add(-1*1*24*time.Hour), accounts[0].AccountID)
	// before cutoff time
	// should get set to lower status
	DB.Exec("UPDATE accounts set updated_at = ? WHERE account_id = ?;", time.Now().Add(-1*3*24*time.Hour), accounts[1].AccountID)

	err := SetAccountsToLowerPaymentStatusByUpdateTime(GasTransferInProgress, time.Now().Add(-1*2*24*time.Hour))
	assert.Nil(t, err)

	accountsFromDB := []Account{}
	DB.Find(&accountsFromDB)

	if accountsFromDB[0].AccountID == accounts[0].AccountID {
		assert.Equal(t, GasTransferInProgress, accountsFromDB[0].PaymentStatus)
	}

	if accountsFromDB[0].AccountID == accounts[1].AccountID {
		assert.Equal(t, InitialPaymentReceived, accountsFromDB[0].PaymentStatus)
	}

	if accountsFromDB[1].AccountID == accounts[0].AccountID {
		assert.Equal(t, GasTransferInProgress, accountsFromDB[1].PaymentStatus)
	}

	if accountsFromDB[1].AccountID == accounts[1].AccountID {
		assert.Equal(t, InitialPaymentReceived, accountsFromDB[1].PaymentStatus)
	}
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
