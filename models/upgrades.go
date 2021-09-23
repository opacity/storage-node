package models

import (
	"encoding/hex"
	"errors"
	"math/big"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

type Upgrade struct {
	/*AccountID associates an entry in the upgrades table with an entry in the accounts table*/
	AccountID       string            `gorm:"primary_key" json:"accountID" validate:"required,len=64"`
	NewStorageLimit StorageLimitType  `gorm:"primary_key;auto_increment:false" json:"newStorageLimit" validate:"required,gte=128" example:"100"` // how much storage they are allowed, in GB.  This will be the new StorageLimit of the account
	OldStorageLimit StorageLimitType  `json:"oldStorageLimit" validate:"required,gte=10" example:"10"`                                           // how much storage they are allowed, in GB.  This will be the new StorageLimit of the account
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
	EthAddress      string            `json:"ethAddress" validate:"required,len=42" minLength:"42" maxLength:"42" example:"a 42-char eth address with 0x prefix"` // the eth address they will send payment to
	EthPrivateKey   string            `json:"ethPrivateKey" validate:"required,len=96"`                                                                           // the private key of the eth address
	PaymentStatus   PaymentStatusType `json:"paymentStatus" validate:"required"`                                                                                  // the status of their payment
	ApiVersion      int               `json:"apiVersion" validate:"omitempty,gte=1" gorm:"default:1"`
	PaymentMethod   PaymentMethodType `json:"paymentMethod" gorm:"default:0"`
	OpctCost        float64           `json:"opctCost" validate:"omitempty,gte=0" example:"1.56"`
	//UsdCost          float64           `json:"usdcost" validate:"omitempty,gte=0" example:"39.99"`
	DurationInMonths int `json:"durationInMonths" gorm:"default:12" validate:"required,gte=1" minimum:"1" example:"12"`
}

/*UpgradeCollectionFunctions maps a PaymentStatus to the method that should be run
on an upgrade of that status*/
var UpgradeCollectionFunctions = make(map[PaymentStatusType]func(
	upgrade Upgrade, networkID uint) error)

func init() {
	UpgradeCollectionFunctions[InitialPaymentInProgress] = handleUpgradeWithPaymentInProgress
	UpgradeCollectionFunctions[InitialPaymentReceived] = handleUpgradeThatNeedsGas
	UpgradeCollectionFunctions[GasTransferInProgress] = handleUpgradeReceivingGas
	UpgradeCollectionFunctions[GasTransferComplete] = handleUpgradeReadyForCollection
	UpgradeCollectionFunctions[PaymentRetrievalInProgress] = handleUpgradeWithCollectionInProgress
	UpgradeCollectionFunctions[PaymentRetrievalComplete] = handleUpgradeAlreadyCollected
}

/*BeforeCreate - callback called before the row is created*/
func (upgrade *Upgrade) BeforeCreate(scope *gorm.Scope) error {
	if upgrade.PaymentStatus < InitialPaymentInProgress {
		upgrade.PaymentStatus = InitialPaymentInProgress
	}
	return utils.Validator.Struct(upgrade)
}

/*BeforeUpdate - callback called before the row is updated*/
func (upgrade *Upgrade) BeforeUpdate(scope *gorm.Scope) error {
	return utils.Validator.Struct(upgrade)
}

/*BeforeDelete - callback called before the row is deleted*/
func (upgrade *Upgrade) BeforeDelete(scope *gorm.Scope) error {
	DeleteStripePaymentIfExists(upgrade.AccountID)
	return nil
}

/*GetOrCreateUpgrade will either get or create an upgrade.  If the upgrade already existed it will update the OpctCost
but will not update the EthAddress and EthPrivateKey*/
func GetOrCreateUpgrade(upgrade Upgrade) (*Upgrade, error) {
	var upgradeFromDB Upgrade

	upgradeFromDB, err := GetUpgradeFromAccountIDAndStorageLimits(upgrade.AccountID, int(upgrade.NewStorageLimit), int(upgrade.OldStorageLimit))
	if len(upgradeFromDB.AccountID) == 0 {
		err = DB.Create(&upgrade).Error
		upgradeFromDB = upgrade
	} else {
		targetTime := time.Now().Add(-60 * time.Minute)
		if targetTime.After(upgradeFromDB.UpdatedAt) {
			upgradeFromDB.OpctCost = upgrade.OpctCost
			//upgradeFromDB.UsdCost = upgrade.UsdCost
			err = DB.Model(&upgradeFromDB).Updates(Upgrade{OpctCost: upgrade.OpctCost}).Error
		}
	}

	return &upgradeFromDB, err
}

/*GetUpgradeFromAccountIDAndStorageLimits will get an upgrade based on AccountID and storage limits*/
func GetUpgradeFromAccountIDAndStorageLimits(accountID string, newStorageLimit, oldStorageLimit int) (Upgrade, error) {
	upgrade := Upgrade{}
	err := DB.Where("account_id = ? AND new_storage_limit = ? AND old_storage_limit = ?",
		accountID,
		newStorageLimit,
		oldStorageLimit).First(&upgrade).Error
	return upgrade, err
}

/*GetUpgradesFromAccountID gets all upgrades that have a particular AccountID*/
func GetUpgradesFromAccountID(accountID string) ([]Upgrade, error) {
	var upgrades []Upgrade
	err := DB.Where("account_id = ?",
		accountID).Find(&upgrades).Error
	return upgrades, err
}

/*SetUpgradesToNextPaymentStatus transitions an upgrade to the next payment status*/
func SetUpgradesToNextPaymentStatus(upgrades []Upgrade) {
	for _, upgrade := range upgrades {
		if upgrade.PaymentStatus == PaymentRetrievalComplete {
			continue
		}
		err := DB.Model(&upgrade).Update("payment_status", getNextPaymentStatus(upgrade.PaymentStatus)).Error
		utils.LogIfError(err, nil)
	}
}

/*CheckIfPaid returns whether the upgrade has been paid for*/
func (upgrade *Upgrade) CheckIfPaid(networkID uint) (bool, error) {
	if upgrade.PaymentStatus >= InitialPaymentReceived {
		return true, nil
	}
	costInWei := upgrade.GetTotalCostInWei()
	paid, err := BackendManager.CheckIfPaid(services.StringToAddress(upgrade.EthAddress),
		costInWei, networkID)
	if paid {
		SetUpgradesToNextPaymentStatus([]Upgrade{*(upgrade)})
	}
	return paid, err
}

/*GetTotalCostInWei gets the total cost in wei for an upgrade*/
func (upgrade *Upgrade) GetTotalCostInWei() *big.Int {
	return services.ConvertToWeiUnit(big.NewFloat(upgrade.OpctCost))
}

/*GetUpgradesByPaymentStatus gets upgrades based on the payment status passed in*/
func GetUpgradesByPaymentStatus(paymentStatus PaymentStatusType) []Upgrade {
	var upgrades []Upgrade
	err := DB.Where("payment_status = ?",
		paymentStatus).Find(&upgrades).Error
	utils.LogIfError(err, nil)
	return upgrades
}

/*handleUpgradeWithPaymentInProgress checks if the user has paid for their upgrade, and if so
sets the upgrade to the next payment status.

Not calling SetUpgradesToNextPaymentStatus here because CheckIfPaid calls it
*/
func handleUpgradeWithPaymentInProgress(upgrade Upgrade, networkID uint) error {
	_, err := upgrade.CheckIfPaid(networkID)
	return err
}

/*handleUpgradeThatNeedsGas sends some ETH to an upgrade that we will later need to collect tokens from and sets the
upgrade's payment status to the next status.*/
func handleUpgradeThatNeedsGas(upgrade Upgrade, networkID uint) error {
	paid, _ := upgrade.CheckIfPaid(networkID)
	var transferErr error
	if paid {
		_, _, _, transferErr = services.EthOpsWrapper.TransferETH(services.EthWrappers[networkID],
			services.EthWrappers[networkID].MainWalletAddress,
			services.EthWrappers[networkID].MainWalletPrivateKey,
			services.StringToAddress(upgrade.EthAddress),
			services.EthWrappers[networkID].DefaultGasForPaymentCollection)
		if transferErr == nil {
			SetUpgradesToNextPaymentStatus([]Upgrade{upgrade})
			return nil
		}
	}
	return transferErr
}

/*handleUpgradeReceivingGas checks whether the gas has arrived and transitions the upgrade to the next payment
status if so.*/
func handleUpgradeReceivingGas(upgrade Upgrade, networkID uint) error {
	ethBalance := services.EthOpsWrapper.GetETHBalance(services.EthWrappers[networkID],
		services.StringToAddress(upgrade.EthAddress))

	if ethBalance.Cmp(big.NewInt(0)) > 0 {
		SetUpgradesToNextPaymentStatus([]Upgrade{upgrade})
	}
	return nil
}

/*handleUpgradeReadyForCollection will attempt to retrieve the tokens from the upgrade's payment address and set the
upgrade's payment status to the next status if there are no errors.*/
func handleUpgradeReadyForCollection(upgrade Upgrade, networkID uint) error {
	tokenBalance := services.EthOpsWrapper.GetTokenBalance(services.EthWrappers[networkID],
		services.StringToAddress(upgrade.EthAddress))
	ethBalance := services.EthOpsWrapper.GetETHBalance(services.EthWrappers[networkID],
		services.StringToAddress(upgrade.EthAddress))

	keyInBytes, decryptErr := utils.DecryptWithErrorReturn(
		utils.Env.EncryptionKey,
		upgrade.EthPrivateKey,
		upgrade.AccountID,
	)
	privateKey, keyErr := services.StringToPrivateKey(hex.EncodeToString(keyInBytes))

	if err := utils.ReturnFirstError([]error{decryptErr, keyErr}); err != nil {
		return err
	} else if tokenBalance.Cmp(big.NewInt(0)) == 0 {
		return errors.New("expected a token balance but found 0")
	} else if ethBalance.Cmp(big.NewInt(0)) == 0 {
		return errors.New("expected an eth balance but found 0")
	} else if tokenBalance.Cmp(big.NewInt(0)) < 0 {
		return errors.New("got negative balance for tokenBalance")
	} else if ethBalance.Cmp(big.NewInt(0)) < 0 {
		return errors.New("got negative balance for ethBalance")
	}

	success, _, _ := services.EthOpsWrapper.TransferToken(services.EthWrappers[networkID],
		services.StringToAddress(upgrade.EthAddress),
		privateKey,
		services.EthWrappers[networkID].MainWalletAddress,
		*tokenBalance,
		services.EthWrappers[networkID].SlowGasPrice)
	if success {
		SetUpgradesToNextPaymentStatus([]Upgrade{upgrade})
		return nil
	}
	return errors.New("payment collection failed")
}

/*handleUpgradeWithCollectionInProgress will check the token balance of an upgrade's payment address.  If the balance
is zero, it means the collection has succeeded and the payment status is set to the next status*/
func handleUpgradeWithCollectionInProgress(upgrade Upgrade, networkID uint) error {
	balance := services.EthOpsWrapper.GetTokenBalance(services.EthWrappers[networkID],
		services.StringToAddress(upgrade.EthAddress))
	if balance.Cmp(big.NewInt(0)) == 0 {
		SetUpgradesToNextPaymentStatus([]Upgrade{upgrade})
	}
	return nil
}

func handleUpgradeAlreadyCollected(upgrade Upgrade, networkID uint) error {
	return nil
}

/*PurgeOldUpgrades deletes upgrades past a certain age*/
func PurgeOldUpgrades(hoursToRetain int) error {
	err := DB.Where("updated_at < ?",
		time.Now().Add(-1*time.Hour*time.Duration(hoursToRetain))).Delete(&Upgrade{}).Error

	return err
}

/*SetUpgradesToLowerPaymentStatusByUpdateTime sets upgrades to a lower payment status if the account has a certain payment
status and the updated_at time is older than the cutoff argument*/
func SetUpgradesToLowerPaymentStatusByUpdateTime(paymentStatus PaymentStatusType, updatedAtCutoffTime time.Time) error {
	return DB.Exec("UPDATE upgrades set payment_status = ? WHERE payment_status = ? AND updated_at < ?", paymentStatus-1, paymentStatus, updatedAtCutoffTime).Error
}
