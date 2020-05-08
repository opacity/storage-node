package models

import (
	"errors"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"math/big"
)

/*StripePayment defines a model for managing a credit card payment*/
type StripePayment struct {
	StripeToken    string          `gorm:"primary_key" json:"stripeToken" binding:"required"`
	AccountID      string          `json:"accountID" binding:"required,len=64"` // some hash of the user's master handle
	ChargeID       string          `json:"chargeID" binding:"omitempty"`
	ApiVersion     int             `json:"apiVersion" binding:"omitempty,gte=1" gorm:"default:1"`
	OpqTxStatus    OpqTxStatusType `json:"opqTxStatus" binding:"required" gorm:"default:1"`
	ChargePaid     bool            `json:"chargePaid"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	UpgradePayment bool            `json:"upgradePayment" gorm:"default:false"`
}

/*OpqTxStatusType defines a type for the OPQ tx statuses*/
type OpqTxStatusType int

const (
	/*OpqTxNotStarted - the opq transaction has not been started*/
	OpqTxNotStarted OpqTxStatusType = iota + 1
	/*OpqTxInProgress - the opq transaction is in progress*/
	OpqTxInProgress
	/*OpqTxSuccess - the opq transaction has finished*/
	OpqTxSuccess
)

const MinutesBeforeRetry = 360

/*OpqTxStatus is for pretty printing the OpqTxStatus*/
var OpqTxStatusMap = make(map[OpqTxStatusType]string)

func init() {
	OpqTxStatusMap[OpqTxNotStarted] = "OpqTxNotStarted"
	OpqTxStatusMap[OpqTxInProgress] = "OpqTxInProgress"
	OpqTxStatusMap[OpqTxSuccess] = "OpqTxSuccess"
}

/*BeforeCreate - callback called before the row is created*/
func (stripePayment *StripePayment) BeforeCreate(scope *gorm.Scope) error {
	if stripePayment.OpqTxStatus < OpqTxNotStarted {
		stripePayment.OpqTxStatus = OpqTxNotStarted
	}

	account, err := GetAccountById(stripePayment.AccountID)
	if err != nil || len(account.AccountID) == 0 {
		return errors.New("cannot create stripe payment for non-existent account")
	}
	if stripePayment.UpgradePayment {
		upgrades, err := GetUpgradesFromAccountID(stripePayment.AccountID)
		if err != nil || len(upgrades) == 0 {
			return errors.New("cannot create stripe payment for upgrade for non-existent upgrade")
		}
	}

	return utils.Validator.Struct(stripePayment)
}

/*GetStripePaymentByAccountId returns Stripe Payment object(first one) if there is not any error. */
func GetStripePaymentByAccountId(accountID string) (StripePayment, error) {
	stripePayment := StripePayment{}
	err := DB.Where("account_id = ?", accountID).First(&stripePayment).Error
	return stripePayment, err
}

/*GetNewestStripePaymentByAccountId returns newest Stripe Payment if there is not any error. */
func GetNewestStripePaymentByAccountId(accountID string) (StripePayment, error) {
	stripePayment := StripePayment{}
	err := DB.Order("created_at desc").Where("account_id = ?", accountID).First(&stripePayment).Error
	return stripePayment, err
}

/*CheckForPaidStripePayment checks for a stripe payment that has been paid. */
func CheckForPaidStripePayment(accountID string) (bool, error) {
	paid := false
	var err error
	stripePayment, err := GetNewestStripePaymentByAccountId(accountID)
	if len(stripePayment.AccountID) != 0 && err == nil {
		paid, err = stripePayment.CheckChargePaid()
	}
	return paid, err
}

/*CheckChargePaid checks if the charge has been paid. */
func (stripePayment *StripePayment) CheckChargePaid() (bool, error) {
	if stripePayment.ChargePaid == true {
		return true, nil
	}
	paid, errStripe := services.CheckChargePaid(stripePayment.ChargeID)
	if !paid {
		return false, errStripe
	}
	errDB := DB.Model(&stripePayment).Update("charge_paid", true).Error
	var errMeta error
	return paid, utils.ReturnFirstError([]error{errStripe, errDB, errMeta})
}

/*SendAccountOPQ sends OPQ to the account associated with a stripe payment. */
func (stripePayment *StripePayment) SendAccountOPQ() error {
	account, err := GetAccountById(stripePayment.AccountID)
	if err != nil {
		return err
	}

	costInWei := account.GetTotalCostInWei()

	success, _, _ := EthWrapper.TransferToken(
		services.MainWalletAddress,
		services.MainWalletPrivateKey,
		services.StringToAddress(account.EthAddress),
		*costInWei,
		services.SlowGasPrice)

	if !success {
		return errors.New("OPQ transaction failed")
	}

	if err := DB.Model(&stripePayment).Update("opq_tx_status", OpqTxInProgress).Error; err != nil {
		return err
	}

	return nil
}

/*SendUpgradeOPQ sends OPQ to the account being upgraded, associated with a stripe payment. */
func (stripePayment *StripePayment) SendUpgradeOPQ(account Account, newStorageLimit int) error {
	upgrade, _ := GetUpgradeFromAccountIDAndStorageLimits(account.AccountID, newStorageLimit, int(account.StorageLimit))

	costInWei := utils.ConvertToWeiUnit(big.NewFloat(upgrade.OpqCost))

	success, _, _ := EthWrapper.TransferToken(
		services.MainWalletAddress,
		services.MainWalletPrivateKey,
		services.StringToAddress(upgrade.EthAddress),
		*costInWei,
		services.SlowGasPrice)

	if !success {
		return errors.New("OPQ transaction failed")
	}

	if err := DB.Model(&stripePayment).Update("opq_tx_status", OpqTxInProgress).Error; err != nil {
		return err
	}

	return nil
}

/*CheckAccountCreationOPQTransaction checks the status of an OPQ payment to an account. */
func (stripePayment *StripePayment) CheckAccountCreationOPQTransaction() (bool, error) {
	account, err := GetAccountById(stripePayment.AccountID)
	if err != nil {
		return false, err
	}

	paid, err := account.CheckIfPaid()
	if err != nil {
		return false, err
	}

	if paid {
		if err := DB.Model(&stripePayment).Update("opq_tx_status", OpqTxSuccess).Error; err != nil {
			return false, err
		}
		return true, err
	}

	err = stripePayment.RetryIfTimedOut()

	return false, err
}

/*CheckUpgradeOPQTransaction checks the status of an OPQ payment to an upgrade. */
func (stripePayment *StripePayment) CheckUpgradeOPQTransaction(account Account, newStorageLimit int) (bool, error) {
	upgrade, err := GetUpgradeFromAccountIDAndStorageLimits(account.AccountID, newStorageLimit, int(account.StorageLimit))
	if err != nil {
		return false, err
	}

	paid, err := upgrade.CheckIfPaid()
	if err != nil {
		return false, err
	}

	if paid {
		if err := DB.Model(&stripePayment).Update("opq_tx_status", OpqTxSuccess).Error; err != nil {
			return false, err
		}
		return true, err
	}

	err = stripePayment.RetryIfTimedOut()

	return false, err
}

/*RetryIfTimedOut retries an OPQ payment to an account if the transaction is timed out. */
func (stripePayment *StripePayment) RetryIfTimedOut() error {
	targetTime := time.Now().Add(-1 * MinutesBeforeRetry * time.Minute)

	if targetTime.After(stripePayment.UpdatedAt) {
		return stripePayment.SendAccountOPQ()
	}
	return nil
}

/*DeleteStripePaymentIfExists deletes a stripe payment if it exists. */
func DeleteStripePaymentIfExists(accountID string) error {
	stripePayment, _ := GetStripePaymentByAccountId(accountID)
	if len(stripePayment.AccountID) != 0 {
		return DB.Delete(&stripePayment).Error
	}
	return nil
}

/*PurgeOldStripePayments deletes stripe payments past a certain age*/
func PurgeOldStripePayments(daysToRetainStripePayment int) error {
	err := DB.Where("updated_at < ? AND opq_tx_status = ?",
		time.Now().Add(-1*time.Hour*24*time.Duration(daysToRetainStripePayment)),
		OpqTxSuccess).Delete(&StripePayment{}).Error

	return err
}
