package models

import (
	"errors"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

/*StripePayment defines a model for managing a credit card payment*/
type StripePayment struct {
	StripeToken string          `gorm:"primary_key" json:"stripeToken" binding:"required"`
	AccountID   string          `json:"accountID" binding:"required,len=64"` // some hash of the user's master handle
	ChargeID    string          `json:"chargeID" binding:"omitempty"`
	ApiVersion  int             `json:"apiVersion" binding:"omitempty,gte=1" gorm:"default:1"`
	OpqTxStatus OpqTxStatusType `json:"opqTxStatus" binding:"required" gorm:"default:1"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
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

	return utils.Validator.Struct(stripePayment)
}

/*Return Stripe Payment object(first one) if there is not any error. */
func GetStripePaymentByAccountId(accountID string) (StripePayment, error) {
	stripePayment := StripePayment{}
	err := DB.Where("account_id = ?", accountID).First(&stripePayment).Error
	return stripePayment, err
}

/*CheckForPaidStripePayment checks for a stripe payment that has been paid. */
func CheckForPaidStripePayment(accountID string) (bool, error) {
	paid := false
	var err error
	stripePayment, err := GetStripePaymentByAccountId(accountID)
	if len(stripePayment.AccountID) != 0 && err == nil {
		paid, err = services.CheckChargePaid(stripePayment.ChargeID)
	}
	if account, _ := GetAccountById(accountID); len(account.AccountID) != 0 && paid && len(account.MetadataKey) != 0 {
		err = HandleMetadataKeyForPaidAccount(account)
	}
	return paid, err
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

/*CheckOPQTransaction checks the status of an OPQ payment to an account. */
func (stripePayment *StripePayment) CheckOPQTransaction() (bool, error) {
	account, err := GetAccountById(stripePayment.AccountID)
	if err != nil {
		return false, err
	}

	paid, err := account.CheckIfPaid()
	if err != nil {
		return false, err
	}

	if paid {
		DB.Delete(stripePayment)
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
