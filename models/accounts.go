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
	"math"
)

const (
	/*PaymentMethodNone as default value.*/
	PaymentMethodNone PaymentMethodType = iota

	/*PaymentMethodWithCreditCard indicated this payment is via Stripe Creditcard processing.*/
	PaymentMethodWithCreditCard
)

/*Account defines a model for managing a user subscription for uploads*/
type Account struct {
	AccountID                string            `gorm:"primary_key" json:"accountID" binding:"required,len=64"` // some hash of the user's master handle
	CreatedAt                time.Time         `json:"createdAt"`
	UpdatedAt                time.Time         `json:"updatedAt"`
	MonthsInSubscription     int               `json:"monthsInSubscription" binding:"required,gte=1" example:"12"`                                                        // number of months in their subscription
	StorageLocation          string            `json:"storageLocation" binding:"omitempty,url"`                                                                           // where their files live, on S3 or elsewhere
	StorageLimit             StorageLimitType  `json:"storageLimit" binding:"required,gte=10" example:"100"`                                                              // how much storage they are allowed, in GB
	StorageUsedInByte        int64             `json:"storageUsedInByte" binding:"exists,gte=0" example:"30"`                                                             // how much storage they have used, in B
	EthAddress               string            `json:"ethAddress" binding:"required,len=42" minLength:"42" maxLength:"42" example:"a 42-char eth address with 0x prefix"` // the eth address they will send payment to
	EthPrivateKey            string            `json:"ethPrivateKey" binding:"required,len=96"`                                                                           // the private key of the eth address
	PaymentStatus            PaymentStatusType `json:"paymentStatus" binding:"required"`                                                                                  // the status of their payment
	ApiVersion               int               `json:"apiVersion" binding:"omitempty,gte=1" gorm:"default:1"`
	TotalFolders             int               `json:"totalFolders" binding:"omitempty,gte=0" gorm:"default:0"`
	TotalMetadataSizeInBytes int64             `json:"totalMetadataSizeInBytes" binding:"omitempty,gte=0" gorm:"default:0"`
	PaymentMethod            PaymentMethodType `json:"paymentMethod" gorm:"default:0"`
}

/*SpaceReport defines a model for capturing the space allotted compared to space used*/
type SpaceReport struct {
	SpaceAllottedSum int
	SpaceUsedSum     float64
}

/*Invoice is the invoice object we will return to the client*/
type Invoice struct {
	Cost       float64 `json:"cost" binding:"omitempty,gte=0" example:"1.56"`
	EthAddress string  `json:"ethAddress" binding:"required,len=42" minLength:"42" maxLength:"42" example:"a 42-char eth address with 0x prefix"`
}

/*StorageLimitType defines a type for the storage limits*/
type StorageLimitType int

/*PaymentStatusType defines a type for the payment statuses*/
type PaymentStatusType int

type PaymentMethodType int

const (
	/*BasicStorageLimit allows 128 GB on the basic plan*/
	BasicStorageLimit StorageLimitType = iota + 128

	/*ProfessionalStorageLimit allows 1024 GB on the professional plan*/
	ProfessionalStorageLimit StorageLimitType = iota + 1024
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

const StorageUsedTooLow = "storage_used_in_byte cannot go below 0"

/*DefaultMonthsPerSubscription is the number of months per year since our
default subscription is a year*/
const DefaultMonthsPerSubscription = 12

/*BasicSubscriptionDefaultCost is the cost for a default-length term of the basic plan*/
const BasicSubscriptionDefaultCost = 2.0

/*AccountIDLength is the expected length of an accountID for an account*/
const AccountIDLength = 64

/*PaymentStatusMap is for pretty printing the PaymentStatus*/
var PaymentStatusMap = make(map[PaymentStatusType]string)

/*PaymentCollectionFunctions maps a PaymentStatus to the method that should be run
on an account of that status*/
var PaymentCollectionFunctions = make(map[PaymentStatusType]func(
	account Account) error)

var InvalidStorageLimitError = errors.New("storage not offered in that increment in GB")

func init() {
	PaymentStatusMap[InitialPaymentInProgress] = "InitialPaymentInProgress"
	PaymentStatusMap[InitialPaymentReceived] = "InitialPaymentReceived"
	PaymentStatusMap[GasTransferInProgress] = "GasTransferInProgress"
	PaymentStatusMap[GasTransferComplete] = "GasTransferComplete"
	PaymentStatusMap[PaymentRetrievalInProgress] = "PaymentRetrievalInProgress"
	PaymentStatusMap[PaymentRetrievalComplete] = "PaymentRetrievalComplete"

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
	if utils.FreeModeEnabled() || utils.Env.Plans[int(account.StorageLimit)].Name == "Free" {
		account.PaymentStatus = PaymentRetrievalComplete
	}
	return utils.Validator.Struct(account)
}

/*BeforeUpdate - callback called before the row is updated*/
func (account *Account) BeforeUpdate(scope *gorm.Scope) error {
	return utils.Validator.Struct(account)
}

/*BeforeDelete - callback called before the row is deleted*/
func (account *Account) BeforeDelete(scope *gorm.Scope) error {
	DeleteStripePaymentIfExists(account.AccountID)
	return nil
}

/*ExpirationDate returns the date the account expires*/
func (account *Account) ExpirationDate() time.Time {
	return account.CreatedAt.AddDate(0, account.MonthsInSubscription, 0)
}

/*Cost returns the expected price of the subscription*/
func (account *Account) Cost() (float64, error) {
	return utils.Env.Plans[int(account.StorageLimit)].Cost *
		float64(account.MonthsInSubscription/DefaultMonthsPerSubscription), nil
}

/*UpgradeCostInOPQ returns the cost to upgrade in OPQ*/
func (account *Account) UpgradeCostInOPQ(upgradeStorageLimit int, monthsForNewPlan int) (float64, error) {
	baseCostOfHigherPlan := utils.Env.Plans[int(upgradeStorageLimit)].Cost *
		float64(monthsForNewPlan/DefaultMonthsPerSubscription)
	costOfCurrentPlan, _ := account.Cost()

	if account.ExpirationDate().Before(time.Now()) {
		return baseCostOfHigherPlan, nil
	}

	return upgradeCost(costOfCurrentPlan, baseCostOfHigherPlan, account)
}

/*UpgradeCostInUSD returns the cost to upgrade in USD*/
func (account *Account) UpgradeCostInUSD(upgradeStorageLimit int, monthsForNewPlan int) (float64, error) {
	baseCostOfHigherPlan := utils.Env.Plans[int(upgradeStorageLimit)].CostInUSD *
		float64(monthsForNewPlan/DefaultMonthsPerSubscription)
	costOfCurrentPlan := utils.Env.Plans[int(account.StorageLimit)].CostInUSD *
		float64(account.MonthsInSubscription/DefaultMonthsPerSubscription)

	if account.ExpirationDate().Before(time.Now()) {
		return baseCostOfHigherPlan, nil
	}

	return upgradeCost(costOfCurrentPlan, baseCostOfHigherPlan, account)
}

func upgradeCost(costOfCurrentPlan, baseCostOfHigherPlan float64, account *Account) (float64, error) {
	accountTimeTotal := account.ExpirationDate().Sub(account.CreatedAt)
	accountTimeRemaining := accountTimeTotal - time.Now().Sub(account.CreatedAt)

	accountBalance := costOfCurrentPlan * float64(accountTimeRemaining.Hours()/accountTimeTotal.Hours())

	paymentStillNeeded := baseCostOfHigherPlan - (accountBalance)

	return math.Ceil(paymentStillNeeded), nil
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
		SetAccountsToNextPaymentStatus([]Account{*(account)})
	}
	return paid, err
}

/*CheckIfPending returns whether a transaction is pending to the address*/
func (account *Account) CheckIfPending() (bool, error) {
	return BackendManager.CheckIfPending(services.StringToAddress(account.EthAddress))
}

/*UseStorageSpaceInByte updates the account's StorageUsedInByte value*/
func (account *Account) UseStorageSpaceInByte(planToUsedInByte int64) error {
	paid, err := account.CheckIfPaid()
	if err != nil {
		return err
	}
	if paidWithCreditCard, _ := CheckForPaidStripePayment(account.AccountID); !paidWithCreditCard && !paid {
		return errors.New("no payment. Unable to update the storage")
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

	plannedInGB := (float64(planToUsedInByte) + float64(accountFromDB.StorageUsedInByte)) / 1e9

	if plannedInGB > float64(accountFromDB.StorageLimit) {
		return errors.New("unable to store more data")
	}

	if err := tx.Model(&accountFromDB).Update("storage_used_in_byte",
		gorm.Expr("storage_used_in_byte + ?", planToUsedInByte)).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Where("account_id = ?", account.AccountID).First(&accountFromDB).Error; err != nil {
		tx.Rollback()
		return err
	}
	if accountFromDB.StorageUsedInByte < int64(0) {
		tx.Rollback()
		return errors.New(StorageUsedTooLow)
	}
	if (accountFromDB.StorageUsedInByte / 1e9) > int64(accountFromDB.StorageLimit) {
		tx.Rollback()
		return errors.New("unable to store more data")
	}

	return tx.Commit().Error
}

/*MaxAllowedMetadataSizeInBytes returns the maximum possible metadata size for an account based on its plan*/
func (account *Account) MaxAllowedMetadataSizeInBytes() int64 {
	maxAllowedMetadataSizeInMB := utils.Env.Plans[int(account.StorageLimit)].MaxMetadataSizeInMB
	return maxAllowedMetadataSizeInMB * 1e6
}

/*MaxAllowedMetadatas returns the maximum possible number of metadatas for an account based on its plan*/
func (account *Account) MaxAllowedMetadatas() int {
	return utils.Env.Plans[int(account.StorageLimit)].MaxFolders
}

/*CanAddNewMetadata checks if an account can have another metadata*/
func (account *Account) CanAddNewMetadata() bool {
	intendedNumberOfMetadatas := account.TotalFolders + 1
	return intendedNumberOfMetadatas <= account.MaxAllowedMetadatas()
}

/*CanRemoveMetadata checks if an account can delete a metadata*/
func (account *Account) CanRemoveMetadata() bool {
	intendedNumberOfMetadatas := account.TotalFolders - 1
	return intendedNumberOfMetadatas >= 0
}

/*CanUpdateMetadata deducts the old size of a metadata, adds the size of the new value the user has sent,
and makes sure the intended total metadata size is below the amount the user is allowed to have*/
func (account *Account) CanUpdateMetadata(oldMetadataSizeInBytes, newMetadataSizeInBytes int64) bool {
	intendedMetadataSizeInBytes := account.TotalMetadataSizeInBytes - oldMetadataSizeInBytes + newMetadataSizeInBytes
	return intendedMetadataSizeInBytes <= account.MaxAllowedMetadataSizeInBytes() &&
		intendedMetadataSizeInBytes >= 0
}

/*UpdatePaymentViaStripe update PaymentMethod to be PaymentMethodWithCreditCard*/
func (account *Account) UpdatePaymentViaStripe() error {
	account.PaymentMethod = PaymentMethodWithCreditCard
	return DB.Model(&account).Update("payment_method", account.PaymentMethod).Error
}

/*IncrementMetadataCount increments the account's metadata count*/
func (account *Account) IncrementMetadataCount() error {
	err := errors.New("cannot exceed allowed metadatas")
	if account.CanAddNewMetadata() {
		account.TotalFolders++
		err = DB.Model(&account).Update("total_folders", account.TotalFolders).Error
	}
	return err
}

/*DecrementMetadataCount decrements the account's metadata count*/
func (account *Account) DecrementMetadataCount() error {
	err := errors.New("metadata count cannot go below 0")
	account.TotalFolders--
	if account.TotalFolders >= 0 {
		err = DB.Model(&account).Update("total_folders", account.TotalFolders).Error
	}
	return err
}

/*UpdateMetadataSizeInBytes updates the account's TotalMetadataSizeInBytes or returns an error*/
func (account *Account) UpdateMetadataSizeInBytes(oldMetadataSizeInBytes, newMetadataSizeInBytes int64) error {
	err := errors.New("metadata size is too large for this account")
	if account.CanUpdateMetadata(oldMetadataSizeInBytes, newMetadataSizeInBytes) {
		account.TotalMetadataSizeInBytes = account.TotalMetadataSizeInBytes - oldMetadataSizeInBytes + newMetadataSizeInBytes
		err = DB.Model(&account).Update("total_metadata_size_in_bytes", account.TotalMetadataSizeInBytes).Error
	}

	return err
}

/*RemoveMetadata removes a metadata and its size from TotalMetadataSizeInBytes*/
func (account *Account) RemoveMetadata(oldMetadataSizeInBytes int64) error {
	err := errors.New("cannot remove metadata or its size")
	if account.CanUpdateMetadata(oldMetadataSizeInBytes, 0) && account.CanRemoveMetadata() {
		account.TotalMetadataSizeInBytes = account.TotalMetadataSizeInBytes - oldMetadataSizeInBytes
		account.TotalFolders--
		err = DB.Model(account).Updates(map[string]interface{}{
			"total_folders":                account.TotalFolders,
			"total_metadata_size_in_bytes": account.TotalMetadataSizeInBytes,
		}).Error
	}
	return err
}

func (account *Account) UpgradeAccount(upgradeStorageLimit int, monthsForNewPlan int) error {
	_, ok := utils.Env.Plans[upgradeStorageLimit]
	if !ok {
		return InvalidStorageLimitError
	}
	if upgradeStorageLimit == int(account.StorageLimit) {
		// assume they have already upgraded and the method has simply been called twice
		return nil
	}
	monthsSinceCreation := differenceInMonths(account.CreatedAt, time.Now())
	account.StorageLimit = StorageLimitType(upgradeStorageLimit)
	account.MonthsInSubscription = monthsSinceCreation + monthsForNewPlan
	return DB.Model(account).Updates(map[string]interface{}{
		"months_in_subscription": account.MonthsInSubscription,
		"storage_limit":          account.StorageLimit,
	}).Error
}

func differenceInMonths(a, b time.Time) int {
	y1, M1, d1 := a.Date()
	y2, M2, d2 := b.Date()

	year := int(y2 - y1)
	day := int(d2 - d1)
	month := int(M2 - M1)

	// Normalize negative values
	if day < 0 {
		// days in month:
		t := time.Date(y1, M1, 32, 0, 0, 0, 0, time.UTC)
		day += 32 - t.Day()
		month--
	}
	if month < 0 {
		month += 12
		year--
	}

	return month + (year * 12) + int(math.Ceil(float64(day)/30))
}

/*Return Account object(first one) if there is not any error. */
func GetAccountById(accountID string) (Account, error) {
	account := Account{}
	err := DB.Where("account_id = ?", accountID).First(&account).Error
	return account, err
}

/*CreateSpaceUsedReport populates a model of the space allotted versus space used*/
func CreateSpaceUsedReport() SpaceReport {
	var result SpaceReport
	DB.Raw("SELECT SUM(storage_limit) as space_allotted_sum, SUM(storage_used_in_byte) as space_used_sum FROM "+
		"accounts WHERE payment_status >= ?", InitialPaymentReceived).Scan(&result)
	return result
}

/*CreateSpaceUsedReportForPlanType populates a model of the space allotted versus space used for a
particular type of plan*/
func CreateSpaceUsedReportForPlanType(storageLimit StorageLimitType) SpaceReport {
	var result SpaceReport
	DB.Raw("SELECT SUM(storage_limit) as space_allotted_sum, SUM(storage_used_in_byte) as space_used_sum FROM "+
		"accounts WHERE payment_status >= ? AND storage_limit = ?",
		InitialPaymentReceived, storageLimit).Scan(&result)
	return result
}

/*CalculatePercentSpaceUsed accepts a space report and calculates the percent of space used vs.
the space allotted*/
func CalculatePercentSpaceUsed(spaceReport SpaceReport) float64 {
	spaceUsedInGB := float64(spaceReport.SpaceUsedSum) / 1e9
	return (spaceUsedInGB / float64(spaceReport.SpaceAllottedSum)) * float64(100)
}

/*PurgeOldUnpaidAccounts deletes accounts past a certain age which have not been paid for*/
func PurgeOldUnpaidAccounts(daysToRetainUnpaidAccounts int) error {
	accounts := []Account{}
	err := DB.Where("created_at < ? AND payment_status = ? AND storage_used_in_byte = ?",
		time.Now().Add(-1*time.Hour*24*time.Duration(daysToRetainUnpaidAccounts)),
		InitialPaymentInProgress,
		int64(0)).Find(&accounts).Error
	for _, account := range accounts {
		if paid, _ := CheckForPaidStripePayment(account.AccountID); !paid {
			err = DB.Delete(&account).Error
		}
	}
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

/*CountPaidAccountsByPlanType counts all paid accounts based on the plan type*/
func CountPaidAccountsByPlanType(storageLimit StorageLimitType) (int, error) {
	count := 0
	err := DB.Model(&Account{}).Where("storage_limit = ? AND payment_status >= ?",
		storageLimit, InitialPaymentReceived).Count(&count).Error
	utils.LogIfError(err, nil)
	return count, err
}

func CountPaidAccountsByPaymentMethodAndPlanType(storageLimit StorageLimitType, paymentMethod PaymentMethodType) (int, error) {
	count := 0
	err := DB.Model(&Account{}).Where("storage_limit = ? AND payment_status >= ? AND payment_method = ?",
		storageLimit, InitialPaymentReceived, paymentMethod).Count(&count).Error
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

/*getNextPaymentStatus returns the next payment status in the sequence*/
func getNextPaymentStatus(paymentStatus PaymentStatusType) PaymentStatusType {
	if paymentStatus == PaymentRetrievalComplete {
		return paymentStatus
	}
	return paymentStatus + 1
}

/*handleAccountWithPaymentInProgress checks if the user has paid for their account, and if so
sets the account to the next payment status.

Not calling SetAccountsToNextPaymentStatus here because CheckIfPaid calls it
*/
func handleAccountWithPaymentInProgress(account Account) error {
	_, err := account.CheckIfPaid()
	return err
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
		return errors.New("expected a token balance but found 0")
	} else if ethBalance.Int64() == 0 {
		return errors.New("expected an eth balance but found 0")
	} else if tokenBalance.Int64() > 0 && ethBalance.Int64() > 0 &&
		utils.ReturnFirstError([]error{decryptErr, keyErr}) == nil {
		success, _, _ := EthWrapper.TransferToken(
			services.StringToAddress(account.EthAddress),
			privateKey,
			services.MainWalletAddress,
			*tokenBalance,
			services.SlowGasPrice)
		if success {
			SetAccountsToNextPaymentStatus([]Account{account})
			return nil
		}
		return errors.New("payment collection failed")
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

	fmt.Print("StorageUsedInByte:                    ")
	fmt.Println(account.StorageUsedInByte)

	fmt.Print("StorageLocation:                ")
	fmt.Println(account.StorageLocation)

	fmt.Print("PaymentStatus:                  ")
	fmt.Println(PaymentStatusMap[account.PaymentStatus])

	fmt.Print("EthAddress:                     ")
	fmt.Println(account.EthAddress)

	fmt.Print("EthPrivateKey:                  ")
	fmt.Println(account.EthPrivateKey)

	fmt.Print("ApiVersion:                     ")
	fmt.Println(account.ApiVersion)

	fmt.Print("TotalFolders:                 ")
	fmt.Println(account.TotalFolders)

	fmt.Print("TotalMetadataSizeInBytes:       ")
	fmt.Println(account.TotalMetadataSizeInBytes)
}
