package models

import (
	"errors"
	"fmt"
	"time"

	"math/big"

	"encoding/hex"

	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

/*Account defines a model for managing a user subscription for uploads*/
type Account struct {
	AccountID            string            `gorm:"primary_key" json:"accountID" binding:"required,len=64"` // some hash of the user's master handle
	CreatedAt            time.Time         `json:"createdAt"`
	UpdatedAt            time.Time         `json:"updatedAt"`
	MonthsInSubscription int               `json:"monthsInSubscription" binding:"required,gte=1" example:"12"`                                                        // number of months in their subscription
	StorageLocation      string            `json:"storageLocation" binding:"omitempty,url"`                                                                           // where their files live, on S3 or elsewhere
	StorageLimit         StorageLimitType  `json:"storageLimit" binding:"required,gte=100" example:"100"`                                                             // how much storage they are allowed, in GB
	StorageUsed          float64           `json:"storageUsed" binding:"exists,gte=0" example:"30"`                                                                   // how much storage they have used, in GB
	EthAddress           string            `json:"ethAddress" binding:"required,len=42" minLength:"42" maxLength:"42" example:"a 42-char eth address with 0x prefix"` // the eth address they will send payment to
	EthPrivateKey        string            `json:"ethPrivateKey" binding:"required,len=96"`                                                                           // the private key of the eth address
	PaymentStatus        PaymentStatusType `json:"paymentStatus" binding:"required"`                                                                                  // the status of their payment
	MetadataKey          string            `json:"metadataKey" binding:"omitempty,len=64"`
	ApiVersion           int               `json:"apiVersion" binding:"omitempty,gte=1" gorm:"default:1"`
}

/*SpaceReport defines a model for capturing the space alloted compared to space used*/
type SpaceReport struct {
	SpaceAllotedSum int
	SpaceUsedSum    float64
}

/*Invoice is the invoice object we will return to the client*/
type Invoice struct {
	Cost       float64 `json:"cost" binding:"required,gte=0" example:"1.56"`
	EthAddress string  `json:"ethAddress" binding:"required,len=42" minLength:"42" maxLength:"42" example:"a 42-char eth address with 0x prefix"`
}

/*StorageLimitType defines a type for the storage limits*/
type StorageLimitType int

/*PaymentStatusType defines a type for the payment statuses*/
type PaymentStatusType int

const (
	/*BasicStorageLimit allows 128 GB on the basic plan*/
	BasicStorageLimit StorageLimitType = iota + 128
)

const (
	/*InitialPaymentInProgress - we have created the subscription and are awaiting their initial payment*/
	InitialPaymentInProgress PaymentStatusType = iota + 1

	/*InitialPaymentReceived - we have received the payment*/
	InitialPaymentReceived

	/*GasTransferInProgress - we are sending gas to the payment address*/
	GasTransferInProgress

	/*GasTransferComplete - our gas transfer is complete*/
	GasTransferComplete

	/*PaymentRetrievalInProgress - we are retrieving their tokens*/
	PaymentRetrievalInProgress

	/*PaymentRetrievalComplete - we have retrieved their tokens*/
	PaymentRetrievalComplete

	// TODO: error states?  Not sure if we need them.
)

/*DefaultMonthsPerSubscription is the number of months per year since our
default subscription is a year*/
const DefaultMonthsPerSubscription = 12

/*BasicSubscriptionDefaultCost is the cost for a default-length term of the basic plan*/
const BasicSubscriptionDefaultCost = 2.0

/*AccountIDLength is the expected length of an accountID for an account*/
const AccountIDLength = 64

/*PaymentStatusMap is for pretty printing the PaymentStatus*/
var PaymentStatusMap = make(map[PaymentStatusType]string)

/*StorageLimitMap maps the amount of storage passed in by the client
to storage limits set by us.  Used for checking that they have passed in an allowed
amount.*/
var StorageLimitMap = make(map[int]StorageLimitType)

/*CostMap is for mapping subscription plans to their prices, for a default length subscription*/
var CostMap = make(map[StorageLimitType]float64)

/*PaymentCollectionFunctions maps a PaymentStatus to the method that should be run
on an account of that status*/
var PaymentCollectionFunctions = make(map[PaymentStatusType]func(
	account Account) error)

func init() {
	StorageLimitMap[int(BasicStorageLimit)] = BasicStorageLimit

	PaymentStatusMap[InitialPaymentInProgress] = "InitialPaymentInProgress"
	PaymentStatusMap[InitialPaymentReceived] = "InitialPaymentReceived"
	PaymentStatusMap[GasTransferInProgress] = "GasTransferInProgress"
	PaymentStatusMap[GasTransferComplete] = "GasTransferComplete"
	PaymentStatusMap[PaymentRetrievalInProgress] = "PaymentRetrievalInProgress"
	PaymentStatusMap[PaymentRetrievalComplete] = "PaymentRetrievalComplete"

	CostMap[BasicStorageLimit] = BasicSubscriptionDefaultCost

	PaymentCollectionFunctions[InitialPaymentInProgress] = handleAccountWithPaymentInProgress
	PaymentCollectionFunctions[InitialPaymentReceived] = handleAccountThatNeedsGas
	PaymentCollectionFunctions[GasTransferInProgress] = handleAccountReceivingGas
	PaymentCollectionFunctions[GasTransferComplete] = handleAccountReadyForCollection
	PaymentCollectionFunctions[PaymentRetrievalInProgress] = handleAccountWithCollectionInProgress
	PaymentCollectionFunctions[PaymentRetrievalComplete] = handleAccountAlreadyCollected
}

/*BeforeCreate - callback called before the row is created*/
func (account *Account) BeforeCreate(scope *gorm.Scope) error {
	if account.PaymentStatus < InitialPaymentInProgress {
		account.PaymentStatus = InitialPaymentInProgress
	}
	return utils.Validator.Struct(account)
}

/*BeforeUpdate - callback called before the row is updated*/
func (account *Account) BeforeUpdate(scope *gorm.Scope) error {
	return utils.Validator.Struct(account)
}

/*ExpirationDate returns the date the account expires*/
func (account *Account) ExpirationDate() time.Time {
	return account.CreatedAt.AddDate(0, account.MonthsInSubscription, 0)
}

/*Cost returns the expected price of the subscription*/
func (account *Account) Cost() (float64, error) {
	costForDefaultSubscriptionTerm, ok := CostMap[account.StorageLimit]
	if !ok {
		return 0, errors.New("no price established for that amount of storage")
	}
	return costForDefaultSubscriptionTerm * float64(account.MonthsInSubscription/DefaultMonthsPerSubscription), nil
}

/*GetTotalCostInWei gets the total cost in wei for a subscription*/
func (account *Account) GetTotalCostInWei() *big.Int {
	float64Cost, _ := account.Cost()
	return utils.ConvertToWeiUnit(big.NewFloat(float64Cost))
}

/*CheckIfPaid returns whether the account has been paid for*/
func (account *Account) CheckIfPaid() (bool, error) {
	if account.PaymentStatus >= InitialPaymentReceived {
		return true, nil
	}
	costInWei := account.GetTotalCostInWei()
	paid, err := BackendManager.CheckIfPaid(services.StringToAddress(account.EthAddress),
		costInWei)
	if paid {
		err := HandleMetadataKeyForPaidAccount(*account)
		if err != nil {
			return false, err
		}
		SetAccountsToNextPaymentStatus([]Account{*(account)})
	}
	return paid, err
}

/*CheckIfPending returns whether a transaction is pending to the address*/
func (account *Account) CheckIfPending() (bool, error) {
	return BackendManager.CheckIfPending(services.StringToAddress(account.EthAddress))
}

/*UseStorageSpaceInByte updates the account's StorageUsed value*/
func (account *Account) UseStorageSpaceInByte(planToUsedInByte int) error {
	paid, err := account.CheckIfPaid()
	if err != nil {
		return err
	}
	if !paid {
		return errors.New("No payment. Unable to update the storage")
	}

	tx := DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	var accountFromDB Account
	if err := tx.Where("account_id = ?", account.AccountID).First(&accountFromDB).Error; err != nil {
		tx.Rollback()
		return err
	}

	inGb := float64(planToUsedInByte) / float64(1e9)
	if inGb+accountFromDB.StorageUsed > float64(accountFromDB.StorageLimit) {
		return errors.New("Unable to store more data")
	}
	updatedStorage := accountFromDB.StorageUsed + inGb

	if err := tx.Model(&accountFromDB).Update("storage_used", updatedStorage).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

/*Return Account object(first one) if there is not any error. */
func GetAccountById(accountID string) (Account, error) {
	account := Account{}
	err := DB.Where("account_id = ?", accountID).First(&account).Error
	return account, err
}

/*CreateSpaceUsedReport populates a model of the space alloted versus space used*/
func CreateSpaceUsedReport() SpaceReport {
	var result SpaceReport
	DB.Raw("SELECT SUM(storage_limit) as space_alloted_sum, SUM(storage_used) as space_used_sum FROM accounts WHERE payment_status >= ?",
		InitialPaymentReceived).Scan(&result)
	return result
}

/*PurgeOldUnpaidAccounts deletes accounts past a certain age which have not been paid for*/
func PurgeOldUnpaidAccounts(daysToRetainUnpaidAccounts int) error {
	err := DB.Where("created_at < ? AND payment_status = ? AND storage_used = ?",
		time.Now().Add(-1*time.Hour*24*time.Duration(daysToRetainUnpaidAccounts)),
		InitialPaymentInProgress,
		float64(0)).Delete(&Account{}).Error
	return err
}

/*getAccountsByPaymentStatus gets accounts based on the payment status passed in*/
func GetAccountsByPaymentStatus(paymentStatus PaymentStatusType) []Account {
	accounts := []Account{}
	err := DB.Where("payment_status = ?",
		paymentStatus).Find(&accounts).Error
	utils.LogIfError(err, nil)
	return accounts
}

/*CountAccountsByPaymentStatus counts accounts based on the payment status passed in*/
func CountAccountsByPaymentStatus(paymentStatus PaymentStatusType) (int, error) {
	count := 0
	err := DB.Model(&Account{}).Where("payment_status = ?",
		paymentStatus).Count(&count).Error
	utils.LogIfError(err, nil)
	return count, err
}

/*SetAccountsToNextPaymentStatus transitions an account to the next payment status*/
func SetAccountsToNextPaymentStatus(accounts []Account) {
	for _, account := range accounts {
		if account.PaymentStatus == PaymentRetrievalComplete {
			continue
		}
		err := DB.Model(&account).Update("payment_status", getNextPaymentStatus(account.PaymentStatus)).Error
		utils.LogIfError(err, nil)
	}
}

/*HandleMetadataKeyForPaidAccount adds the metadata key to badger and removes from the sql table*/
func HandleMetadataKeyForPaidAccount(account Account) (err error) {
	// Create empty key:value data in badger DB
	ttl := time.Until(account.ExpirationDate())
	if err = utils.BatchSet(&utils.KVPairs{account.MetadataKey: ""}, ttl); err != nil {
		return err
	}
	// Delete the metadata key on the account model
	if err = DB.Model(&account).Update("metadata_key", "").Error; err != nil {
		return err
	}
	return nil
}

/*getNextPaymentStatus returns the next payment status in the sequence*/
func getNextPaymentStatus(paymentStatus PaymentStatusType) PaymentStatusType {
	if paymentStatus == PaymentRetrievalComplete {
		return paymentStatus
	}
	return paymentStatus + 1
}

/*handleAccountWithPaymentInProgress checks if the user has paid for their account, and if so
sets the account to the next payment status, adds the metadata key to badger, and deletes the
metadata key from the SQL DB.

Not calling SetAccountsToNextPaymentStatus here because CheckIfPaid calls it
*/
func handleAccountWithPaymentInProgress(account Account) error {
	_, err := account.CheckIfPaid()
	if err != nil {
		return err
	}

	return nil
}

/*handleAccountThatNeedsGas sends some ETH to an account that we will later need to collect tokens from and sets the
account's payment status to the next status.*/
func handleAccountThatNeedsGas(account Account) error {
	paid, _ := account.CheckIfPaid()
	var transferErr error
	if paid {
		_, _, _, transferErr = EthWrapper.TransferETH(
			services.MainWalletAddress,
			services.MainWalletPrivateKey,
			services.StringToAddress(account.EthAddress),
			services.DefaultGasForPaymentCollection)
		if transferErr == nil {
			SetAccountsToNextPaymentStatus([]Account{account})
			return nil
		}
		utils.LogIfError(transferErr, nil)
	}
	return transferErr
}

/*handleAccountReceivingGas checks whether the gas has arrived and transitions the account to the next payment
status if so.*/
func handleAccountReceivingGas(account Account) error {
	ethBalance := EthWrapper.GetETHBalance(services.StringToAddress(account.EthAddress))
	if ethBalance.Int64() > 0 {
		SetAccountsToNextPaymentStatus([]Account{account})
	}
	return nil
}

/*handleAccountReadyForCollection will attempt to retrieve the tokens from the account's payment address and set the
account's payment status to the next status if there are no errors.*/
func handleAccountReadyForCollection(account Account) error {
	tokenBalance := EthWrapper.GetTokenBalance(services.StringToAddress(account.EthAddress))
	ethBalance := EthWrapper.GetETHBalance(services.StringToAddress(account.EthAddress))
	keyInBytes, decryptErr := utils.DecryptWithErrorReturn(
		utils.Env.EncryptionKey,
		account.EthPrivateKey,
		account.AccountID,
	)
	privateKey, keyErr := services.StringToPrivateKey(hex.EncodeToString(keyInBytes))

	if tokenBalance.Int64() == 0 {
		utils.LogIfError(errors.New("expected a token balance but found 0"), nil)
	} else if ethBalance.Int64() == 0 {
		utils.LogIfError(errors.New("expected an eth balance but found 0"), nil)
	} else if tokenBalance.Int64() > 0 && ethBalance.Int64() > 0 &&
		utils.ReturnFirstError([]error{decryptErr, keyErr}) == nil {
		success, _, _ := EthWrapper.TransferToken(
			services.StringToAddress(account.EthAddress),
			privateKey,
			services.MainWalletAddress,
			*tokenBalance)
		if success {
			SetAccountsToNextPaymentStatus([]Account{account})
			return nil
		}
		utils.LogIfError(errors.New("payment collection failed"), nil)
	}
	return utils.ReturnFirstError([]error{decryptErr, keyErr})
}

/*handleAccountWithCollectionInProgress will check the token balance of an account's payment address.  If the balance
is zero, it means the collection has succeeded and the payment status is set to the next status*/
func handleAccountWithCollectionInProgress(account Account) error {
	balance := EthWrapper.GetTokenBalance(services.StringToAddress(account.EthAddress))
	if balance.Int64() > 0 {
		SetAccountsToNextPaymentStatus([]Account{account})
	}
	return nil
}

/*handleAccountAlreadyCollected does not do anything important and is here to prevent index out of range errors*/
func handleAccountAlreadyCollected(account Account) error {
	return nil
}

/*PrettyString - print the account in a friendly way.  Not used for external logging, just for watching in the
terminal*/
func (account *Account) PrettyString() {
	fmt.Print("AccountID:                      ")
	fmt.Println(account.AccountID)

	fmt.Print("CreatedAt:                      ")
	fmt.Println(account.CreatedAt)

	fmt.Print("UpdatedAt:                      ")
	fmt.Println(account.UpdatedAt)

	fmt.Print("ExpirationDate:                 ")
	fmt.Println(account.ExpirationDate())

	fmt.Print("MonthsInSubscription:           ")
	fmt.Println(account.MonthsInSubscription)

	fmt.Print("Cost:                           ")
	fmt.Println(account.Cost())

	fmt.Print("StorageLimit:                   ")
	fmt.Println(account.StorageLimit)

	fmt.Print("StorageUsed:                    ")
	fmt.Println(account.StorageUsed)

	fmt.Print("StorageLocation:                ")
	fmt.Println(account.StorageLocation)

	fmt.Print("PaymentStatus:                  ")
	fmt.Println(PaymentStatusMap[account.PaymentStatus])

	fmt.Print("EthAddress:                     ")
	fmt.Println(account.EthAddress)

	fmt.Print("EthPrivateKey:                  ")
	fmt.Println(account.EthPrivateKey)
}
