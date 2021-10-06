package models

import (
	"errors"
	"time"

	"math/big"

	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

/*StripePayment defines a model for managing a credit card payment*/
type StripePayment struct {
	StripeToken    string           `gorm:"primary_key" json:"stripeToken" validate:"required"`
	AccountID      string           `json:"accountID" validate:"required,len=64"` // some hash of the user's master handle
	ChargeID       string           `json:"chargeID" validate:"omitempty"`
	ApiVersion     int              `json:"apiVersion" validate:"omitempty,gte=1" gorm:"default:1"`
	OpctTxStatus   OpctTxStatusType `json:"opctTxStatus" validate:"required" gorm:"default:1"`
	ChargePaid     bool             `json:"chargePaid"`
	CreatedAt      time.Time        `json:"createdAt"`
	UpdatedAt      time.Time        `json:"updatedAt"`
	UpgradePayment bool             `json:"upgradePayment" gorm:"default:false"`
}

/*OpctTxStatusType defines a type for the OPCT tx statuses*/
type OpctTxStatusType int

const (
	/*OpctTxNotStarted - the opct transaction has not been started*/
	OpctTxNotStarted OpctTxStatusType = iota + 1
	/*OpctTxInProgress - the opct transaction is in progress*/
	OpctTxInProgress
	/*OpctTxSuccess - the opct transaction has finished*/
	OpctTxSuccess
)

const MinutesBeforeRetry = 360

/*OpctTxStatus is for pretty printing the OpctTxStatus*/
var OpctTxStatusMap = make(map[OpctTxStatusType]string)

func init() {
	OpctTxStatusMap[OpctTxNotStarted] = "OpctTxNotStarted"
	OpctTxStatusMap[OpctTxInProgress] = "OpctTxInProgress"
	OpctTxStatusMap[OpctTxSuccess] = "OpctTxSuccess"
}

/*BeforeCreate - callback called before the row is created*/
func (stripePayment *StripePayment) BeforeCreate(scope *gorm.Scope) error {
	if stripePayment.OpctTxStatus < OpctTxNotStarted {
		stripePayment.OpctTxStatus = OpctTxNotStarted
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
	if stripePayment.ChargePaid {
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

/*SendAccountOPCT sends OPCT to the account associated with a stripe payment. */
func (stripePayment *StripePayment) SendAccountOPCT(networkID uint) error {
	account, err := GetAccountById(stripePayment.AccountID)
	if err != nil {
		return err
	}

	costInWei := account.GetTotalCostInWei()

	success, _, _ := services.EthOpsWrapper.TransferToken(services.EthWrappers[networkID],
		services.EthWrappers[networkID].MainWalletAddress,
		services.EthWrappers[networkID].MainWalletPrivateKey,
		services.StringToAddress(account.EthAddress),
		*costInWei,
		services.EthWrappers[networkID].SlowGasPrice)

	if !success {
		return errors.New("OPCT transaction failed")
	}

	if err := DB.Model(&stripePayment).Update("opct_tx_status", OpctTxInProgress).Error; err != nil {
		return err
	}

	return nil
}

/*SendUpgradeOPCT sends OPCT to the account being upgraded, associated with a stripe payment. */
func (stripePayment *StripePayment) SendUpgradeOPCT(account Account, planID uint, networkID uint) error {
	upgrade, _ := GetUpgradeFromAccountIDAndPlans(account.AccountID, planID, account.PlanInfo.ID)

	costInWei := services.ConvertToWeiUnit(big.NewFloat(upgrade.OpctCost))

	success, _, _ := services.EthOpsWrapper.TransferToken(services.EthWrappers[networkID],
		services.EthWrappers[networkID].MainWalletAddress,
		services.EthWrappers[networkID].MainWalletPrivateKey,
		services.StringToAddress(upgrade.EthAddress),
		*costInWei,
		services.EthWrappers[networkID].SlowGasPrice)

	if !success {
		return errors.New("OPCT transaction failed")
	}

	if err := DB.Model(&stripePayment).Update("opct_tx_status", OpctTxInProgress).Error; err != nil {
		return err
	}

	return nil
}

/*CheckAccountCreationOPCTTransaction checks the status of an OPCT payment to an account. */
func (stripePayment *StripePayment) CheckAccountCreationOPCTTransaction() (bool, error) {
	account, err := GetAccountById(stripePayment.AccountID)
	if err != nil {
		return false, err
	}

	paid, networkID, err := account.CheckIfPaid()
	if err != nil {
		return false, err
	}

	if paid {
		if err := DB.Model(&stripePayment).Update("opct_tx_status", OpctTxSuccess).Error; err != nil {
			return false, err
		}
		return true, err
	}

	err = stripePayment.RetryIfTimedOut(networkID)

	return false, err
}

/*CheckUpgradeOPCTTransaction checks the status of an OPCT payment to an upgrade. */
func (stripePayment *StripePayment) CheckUpgradeOPCTTransaction(account Account, planID uint) (bool, error) {
	upgrade, err := GetUpgradeFromAccountIDAndPlans(account.AccountID, planID, account.PlanInfo.ID)
	if err != nil {
		return false, err
	}

	paid, networkID, err := upgrade.CheckIfPaid()
	if err != nil {
		return false, err
	}

	if paid {
		if err := DB.Model(&stripePayment).Update("opct_tx_status", OpctTxSuccess).Error; err != nil {
			return false, err
		}
		return true, err
	}

	err = stripePayment.RetryIfTimedOut(networkID)

	return false, err
}

/*RetryIfTimedOut retries an OPCT payment to an account if the transaction is timed out. */
func (stripePayment *StripePayment) RetryIfTimedOut(networkID uint) error {
	targetTime := time.Now().Add(-1 * MinutesBeforeRetry * time.Minute)

	if targetTime.After(stripePayment.UpdatedAt) {
		return stripePayment.SendAccountOPCT(networkID)
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
	err := DB.Where("updated_at < ? AND opct_tx_status = ?",
		time.Now().Add(-1*time.Hour*24*time.Duration(daysToRetainStripePayment)),
		OpctTxSuccess).Delete(&StripePayment{}).Error

	return err
}
