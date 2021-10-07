package models

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

const (
	/*PaymentMethodNone as default value.*/
	PaymentMethodNone PaymentMethodType = iota

	/*PaymentMethodWithCreditCard indicated this payment is via Stripe Creditcard processing.*/
	PaymentMethodWithCreditCard
)

/*Account defines a model for managing a user subscription for uploads*/
type Account struct {
	AccountID                string            `gorm:"primary_key" json:"accountID" validate:"required,len=64"` // some hash of the user's master handle
	CreatedAt                time.Time         `json:"createdAt"`
	UpdatedAt                time.Time         `json:"updatedAt"`
	MonthsInSubscription     int               `json:"monthsInSubscription" validate:"required,gte=1" example:"12"`                                                        // number of months in their subscription                                                           // how much storage they are allowed, in GB
	StorageUsedInByte        int64             `json:"storageUsedInByte" validate:"gte=0" example:"30"`                                                                    // how much storage they have used, in B
	EthAddress               string            `json:"ethAddress" validate:"required,len=42" minLength:"42" maxLength:"42" example:"a 42-char eth address with 0x prefix"` // the eth address they will send payment to
	EthPrivateKey            string            `json:"ethPrivateKey" validate:"required,len=96"`                                                                           // the private key of the eth address
	PaymentStatus            PaymentStatusType `json:"paymentStatus" validate:"required"`                                                                                  // the status of their payment
	ApiVersion               int               `json:"apiVersion" validate:"omitempty,gte=1" gorm:"default:2"`
	TotalFolders             int               `json:"totalFolders" validate:"omitempty,gte=0" gorm:"default:0"`
	TotalMetadataSizeInBytes int64             `json:"totalMetadataSizeInBytes" validate:"omitempty,gte=0" gorm:"default:0"`
	PaymentMethod            PaymentMethodType `json:"paymentMethod" gorm:"default:0"`
	Upgrades                 []Upgrade         `gorm:"foreignkey:AccountID;association_foreignkey:AccountID"`
	ExpiredAt                time.Time         `json:"expiredAt"`
	PlanInfoID               uint              `json:"-"`
	PlanInfo                 utils.PlanInfo    `gorm:"foreignKey:plan_info_id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"plan"`
	NetworkIdPaid            uint              `json:"networkIdPaid"`
}

/*SpaceReport defines a model for capturing the space allotted compared to space used*/
type SpaceReport struct {
	SpaceAllottedSum int
	SpaceUsedSum     float64
}

/*Invoice is the invoice object we will return to the client*/
type Invoice struct {
	Cost       float64 `json:"cost" validate:"omitempty,gte=0" example:"1.56"`
	EthAddress string  `json:"ethAddress" validate:"required,len=42" minLength:"42" maxLength:"42" example:"a 42-char eth address with 0x prefix"`
}

/*PaymentStatusType defines a type for the payment statuses*/
type PaymentStatusType int

type PaymentMethodType int

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
// @TODO: Remove this on monthly payments
const DefaultMonthsPerSubscription = 12

/*AccountIDLength is the expected length of an accountID for an account*/
const AccountIDLength = 64

/*PaymentStatusMap is for pretty printing the PaymentStatus*/
var PaymentStatusMap = make(map[PaymentStatusType]string)

/*AccountCollectionFunctions maps a PaymentStatus to the method that should be run
on an account of that status*/
var AccountCollectionFunctions = make(map[PaymentStatusType]func(
	account Account) error)

func init() {
	PaymentStatusMap[InitialPaymentInProgress] = "InitialPaymentInProgress"
	PaymentStatusMap[InitialPaymentReceived] = "InitialPaymentReceived"
	PaymentStatusMap[GasTransferInProgress] = "GasTransferInProgress"
	PaymentStatusMap[GasTransferComplete] = "GasTransferComplete"
	PaymentStatusMap[PaymentRetrievalInProgress] = "PaymentRetrievalInProgress"
	PaymentStatusMap[PaymentRetrievalComplete] = "PaymentRetrievalComplete"

	AccountCollectionFunctions[InitialPaymentInProgress] = handleAccountWithPaymentInProgress
	AccountCollectionFunctions[InitialPaymentReceived] = handleAccountThatNeedsGas
	AccountCollectionFunctions[GasTransferInProgress] = handleAccountReceivingGas
	AccountCollectionFunctions[GasTransferComplete] = handleAccountReadyForCollection
	AccountCollectionFunctions[PaymentRetrievalInProgress] = handleAccountWithCollectionInProgress
	AccountCollectionFunctions[PaymentRetrievalComplete] = handleAccountAlreadyCollected
}

/*BeforeCreate - callback called before the row is created*/
func (account *Account) BeforeCreate(scope *gorm.Scope) error {
	if account.PaymentStatus < InitialPaymentInProgress {
		account.PaymentStatus = InitialPaymentInProgress
	}
	if utils.FreeModeEnabled() || CheckPlanInfoIsFree(account.PlanInfo) {
		account.PaymentStatus = PaymentRetrievalComplete
	}
	account.ExpiredAt = time.Now().AddDate(0, int(account.PlanInfo.MonthsInSubscription), 0)
	return utils.Validator.Struct(account)
}

/*BeforeUpdate - callback called before the row is updated*/
func (account *Account) BeforeUpdate(scope *gorm.Scope) error {
	// @TODO: investigate why ExpiredAt is updated on each update
	account.ExpiredAt = time.Now().AddDate(0, account.MonthsInSubscription, 0)
	return utils.Validator.Struct(account)
}

/*BeforeDelete - callback called before the row is deleted*/
func (account *Account) BeforeDelete(scope *gorm.Scope) error {
	DeleteStripePaymentIfExists(account.AccountID)
	return nil
}

func (account *Account) AfterFind(tx *gorm.DB) (err error) {
	if account.NetworkIdPaid == 0 {
		// Support legacy accounts
		account.NetworkIdPaid = 1
	}
	return
}

/*ExpirationDate returns the date the account expires*/
func (account *Account) ExpirationDate() time.Time {
	// @TODO: Investigate why is the expiration date updated on each call :| possible bug
	account.ExpiredAt = account.CreatedAt.AddDate(0, account.MonthsInSubscription, 0)
	err := DB.Model(&account).Update("expired_at", account.ExpiredAt).Error
	utils.LogIfError(err, nil)
	return account.ExpiredAt
}

/*Cost returns the expected price of the subscription*/
func (account *Account) Cost() (float64, error) {
	return account.PlanInfo.Cost *
		float64(account.MonthsInSubscription/int(account.PlanInfo.MonthsInSubscription)), nil
}

/*UpgradeCostInOPCT returns the cost to upgrade in OPCT*/
func (account *Account) UpgradeCostInOPCT(newPlanInfo utils.PlanInfo) (float64, error) {
	baseCostOfHigherPlan := newPlanInfo.Cost *
		float64(newPlanInfo.MonthsInSubscription/DefaultMonthsPerSubscription)
	costOfCurrentPlan, _ := account.Cost()

	if account.ExpirationDate().Before(time.Now()) {
		return baseCostOfHigherPlan, nil
	}

	return upgradeCost(costOfCurrentPlan, baseCostOfHigherPlan, account)
}

/*UpgradeCostInUSD returns the cost to upgrade in USD*/
func (account *Account) UpgradeCostInUSD(newPlanInfo utils.PlanInfo) (float64, error) {
	baseCostOfHigherPlan := newPlanInfo.CostInUSD *
		float64(newPlanInfo.MonthsInSubscription/DefaultMonthsPerSubscription)
	costOfCurrentPlan := account.PlanInfo.CostInUSD *
		float64(account.MonthsInSubscription/DefaultMonthsPerSubscription)

	if account.ExpirationDate().Before(time.Now()) {
		return baseCostOfHigherPlan, nil
	}

	return upgradeCost(costOfCurrentPlan, baseCostOfHigherPlan, account)
}

func upgradeCost(costOfCurrentPlan, baseCostOfHigherPlan float64, account *Account) (float64, error) {
	accountTimeTotal := account.ExpirationDate().Sub(account.CreatedAt)
	accountTimeRemaining := accountTimeTotal - time.Since(account.CreatedAt)

	accountBalance := costOfCurrentPlan * float64(accountTimeRemaining.Hours()/accountTimeTotal.Hours())

	paymentStillNeeded := baseCostOfHigherPlan - (accountBalance)

	return math.Ceil(paymentStillNeeded), nil
}

/*GetTotalCostInWei gets the total cost in wei for a subscription*/
func (account *Account) GetTotalCostInWei() *big.Int {
	float64Cost, _ := account.Cost()
	return services.ConvertToWeiUnit(big.NewFloat(float64Cost))
}

/*CheckIfPaid returns whether the account has been paid for*/
func (account *Account) CheckIfPaid() (bool, uint, error) {
	if account.PaymentStatus >= InitialPaymentReceived {
		return true, account.NetworkIdPaid, nil
	}

	costInWei := account.GetTotalCostInWei()
	paid, networkID, err := BackendManager.CheckIfPaid(services.StringToAddress(account.EthAddress), costInWei)

	if paid {
		SetAccountsToNextPaymentStatus([]Account{*(account)})
	}
	return paid, networkID, err
}

/*CheckIfPending returns whether a transaction is pending to the address*/
func (account *Account) CheckIfPending() bool {
	return BackendManager.CheckIfPending(services.StringToAddress(account.EthAddress))
}

/*UseStorageSpaceInByte updates the account's StorageUsedInByte value*/
func (account *Account) UseStorageSpaceInByte(planToUsedInByte int64) error {
	paid, _, err := account.CheckIfPaid()
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
	if err := tx.Preload("PlanInfo").Where("account_id = ?", account.AccountID).First(&accountFromDB).Error; err != nil {
		tx.Rollback()
		return err
	}

	plannedInGB := (float64(planToUsedInByte) + float64(accountFromDB.StorageUsedInByte)) / 1e9

	if plannedInGB > float64(accountFromDB.PlanInfo.StorageInGB) {
		return errors.New("unable to store more data")
	}

	if err := tx.Model(&accountFromDB).Update("storage_used_in_byte",
		gorm.Expr("storage_used_in_byte + ?", planToUsedInByte)).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Preload("PlanInfo").Where("account_id = ?", account.AccountID).First(&accountFromDB).Error; err != nil {
		tx.Rollback()
		return err
	}
	if accountFromDB.StorageUsedInByte < int64(0) {
		tx.Rollback()
		return errors.New(StorageUsedTooLow)
	}
	if (accountFromDB.StorageUsedInByte / 1e9) > int64(accountFromDB.PlanInfo.StorageInGB) {
		tx.Rollback()
		return errors.New("unable to store more data")
	}

	return tx.Commit().Error
}

/*MaxAllowedMetadataSizeInBytes returns the maximum possible metadata size for an account based on its plan*/
func (account *Account) MaxAllowedMetadataSizeInBytes() int64 {
	return account.PlanInfo.MaxMetadataSizeInMB * 1e6
}

/*MaxAllowedMetadatas returns the maximum possible number of metadatas for an account based on its plan*/
func (account *Account) MaxAllowedMetadatas() int {
	return account.PlanInfo.MaxFolders
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

/*CanRemoveMetadataMultiple checks if an account can delete multiple metadatas*/
func (account *Account) CanRemoveMetadataMultiple(numberOfMetadatas int) bool {
	intendedNumberOfMetadatas := account.TotalFolders - numberOfMetadatas
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

/*RemoveMetadata removes multiple metadata and their size from TotalMetadataSizeInBytes*/
func (account *Account) RemoveMetadataMultiple(oldMetadataSizeInBytes int64, numberOfMetadatas int) error {
	err := errors.New("cannot remove metadata or its size")
	if account.CanUpdateMetadata(oldMetadataSizeInBytes, 0) && account.CanRemoveMetadataMultiple(numberOfMetadatas) {
		account.TotalMetadataSizeInBytes = account.TotalMetadataSizeInBytes - oldMetadataSizeInBytes
		account.TotalFolders -= numberOfMetadatas
		err = DB.Model(account).Updates(map[string]interface{}{
			"total_folders":                account.TotalFolders,
			"total_metadata_size_in_bytes": account.TotalMetadataSizeInBytes,
		}).Error
	}
	return err
}

// func (account *Account) UpgradeAccount(upgradeStorageLimit int, monthsForNewPlan int) error {
func (account *Account) UpgradeAccount(newPlanInfo utils.PlanInfo) error {
	if newPlanInfo.ID == account.PlanInfo.ID {
		// assume they have already upgraded and the method has simply been called twice
		return nil
	}

	monthsSinceCreation := differenceInMonths(account.CreatedAt, time.Now())

	account.MonthsInSubscription = monthsSinceCreation + int(newPlanInfo.MonthsInSubscription)
	return DB.Model(account).Updates(map[string]interface{}{
		"months_in_subscription": account.MonthsInSubscription,
		"plan_info_id":           newPlanInfo.ID,
		"expired_at":             account.CreatedAt.AddDate(0, account.MonthsInSubscription, 0),
		"updated_at":             time.Now(),
	}).Error
}

func (account *Account) RenewAccount() error {
	return DB.Model(account).Updates(map[string]interface{}{
		"months_in_subscription": account.MonthsInSubscription + int(account.PlanInfo.MonthsInSubscription),
		"expired_at":             account.CreatedAt.AddDate(0, account.MonthsInSubscription+int(account.PlanInfo.MonthsInSubscription), 0),
		"updated_at":             time.Now(),
	}).Error
}

func (account *Account) UpdateNetworkIdPaid(networkID uint) error {
	return DB.Model(&account).Update("network_id_paid", networkID).Error
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
	err := DB.Preload("PlanInfo").Where("account_id = ?", accountID).First(&account).Error
	return account, err
}

/*CreateSpaceUsedReport populates a model of the space allotted versus space used*/
func CreateSpaceUsedReport() SpaceReport {
	var result SpaceReport
	DB.Raw("SELECT SUM(plan_infos.storage_in_gb) as space_allotted_sum, SUM(accounts.storage_used_in_byte) as space_used_sum "+
		"FROM .accounts "+
		"INNER JOIN plan_infos ON plan_infos.id = accounts.plan_info_id "+
		"WHERE accounts.payment_status >= ?",
		InitialPaymentReceived).Scan(&result)
	return result
}

/*CreateSpaceUsedReportForPlanType populates a model of the space allotted versus space used for a
particular type of plan*/
func CreateSpaceUsedReportForPlanType(planInfo utils.PlanInfo) SpaceReport {
	// @TODO: fix this for metrics
	var result SpaceReport
	DB.Raw("SELECT SUM(plan_infos.storage_in_gb) as space_allotted_sum, SUM(accounts.storage_used_in_byte) as space_used_sum "+
		"FROM .accounts "+
		"INNER JOIN plan_infos ON plan_infos.id = accounts.plan_info_id "+
		"WHERE accounts.payment_status >= ? AND plan_infos.id = ?",
		InitialPaymentReceived, planInfo.ID).Scan(&result)
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

/*GetAccountsByPaymentStatus gets accounts based on the payment status passed in*/
func GetAccountsByPaymentStatus(paymentStatus PaymentStatusType) []Account {
	accounts := []Account{}
	err := DB.Preload("PlanInfo").Where("payment_status = ?",
		paymentStatus).Find(&accounts).Error
	utils.LogIfError(err, nil)
	return accounts
}

/*CountAccountsByPaymentStatus counts accounts based on the payment status passed in*/
func CountAccountsByPaymentStatus(paymentStatus PaymentStatusType) (int, error) {
	count := 0
	err := DB.Preload("PlanInfo").Model(&Account{}).Where("payment_status = ?",
		paymentStatus).Count(&count).Error
	utils.LogIfError(err, nil)
	return count, err
}

/*CountPaidAccountsByPlanType counts all paid accounts based on the plan type*/
func CountPaidAccountsByPlanType(planInfo utils.PlanInfo) (int, error) {
	// @TODO: fix this for metrics
	count := 0
	err := DB.Preload("PlanInfo").Model(&Account{}).Where("plan_info_id = ? AND payment_status >= ?",
		planInfo.ID, InitialPaymentReceived).Count(&count).Error
	utils.LogIfError(err, nil)
	return count, err
}

func CountPaidAccountsByPaymentMethodAndPlanType(planInfo utils.PlanInfo, paymentMethod PaymentMethodType) (int, error) {
	// @TODO: fix this for metrics
	count := 0
	err := DB.Preload("PlanInfo").Model(&Account{}).Where("plan_info_id = ? AND payment_status >= ? AND payment_method = ?",
		planInfo.ID, InitialPaymentReceived, paymentMethod).Count(&count).Error
	utils.LogIfError(err, nil)
	return count, err
}

/*SetAccountsToNextPaymentStatus transitions an account to the next payment status*/
func SetAccountsToNextPaymentStatus(accounts []Account) {
	for _, account := range accounts {
		if account.PaymentStatus == PaymentRetrievalComplete {
			continue
		}
		err := DB.Preload("PlanInfo").Model(&account).Update("payment_status", getNextPaymentStatus(account.PaymentStatus)).Error
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
	_, _, err := account.CheckIfPaid()
	return err
}

/*handleAccountThatNeedsGas sends some ETH to an account that we will later need to collect tokens from and sets the
account's payment status to the next status.*/
func handleAccountThatNeedsGas(account Account) error {
	paid, networkID, _ := account.CheckIfPaid()
	var transferErr error
	if paid {
		_, _, _, transferErr = services.EthOpsWrapper.TransferETH(services.EthWrappers[networkID],
			services.EthWrappers[networkID].MainWalletAddress,
			services.EthWrappers[networkID].MainWalletPrivateKey,
			services.StringToAddress(account.EthAddress),
			services.EthWrappers[networkID].DefaultGasForPaymentCollection)
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
	for networkID := range services.EthWrappers {
		ethBalance := services.EthOpsWrapper.GetETHBalance(services.EthWrappers[networkID],
			services.StringToAddress(account.EthAddress))

		if ethBalance.Cmp(big.NewInt(0)) > 0 {
			SetAccountsToNextPaymentStatus([]Account{account})
			return nil
		}
	}

	return nil
}

/*handleAccountReadyForCollection will attempt to retrieve the tokens from the account's payment address and set the
account's payment status to the next status if there are no errors.*/
func handleAccountReadyForCollection(account Account) error {
	for networkID := range services.EthWrappers {
		tokenBalance := services.EthOpsWrapper.GetTokenBalance(services.EthWrappers[networkID],
			services.StringToAddress(account.EthAddress))
		ethBalance := services.EthOpsWrapper.GetETHBalance(services.EthWrappers[networkID],
			services.StringToAddress(account.EthAddress))

		keyInBytes, decryptErr := utils.DecryptWithErrorReturn(
			utils.Env.EncryptionKey,
			account.EthPrivateKey,
			account.AccountID,
		)
		privateKey, keyErr := services.StringToPrivateKey(hex.EncodeToString(keyInBytes))

		if err := utils.ReturnFirstError([]error{decryptErr, keyErr}); err != nil {
			return err
		} else if tokenBalance.Cmp(big.NewInt(0)) == 0 {
			fmt.Printf("expected a token balance but found 0 for networkID %d", networkID)
		} else if ethBalance.Cmp(big.NewInt(0)) == 0 {
			fmt.Printf("expected an eth balance but found 0 for networkID %d", networkID)
		} else if tokenBalance.Cmp(big.NewInt(0)) < 0 {
			fmt.Printf("got negative balance for tokenBalance for networkID %d", networkID)
		} else if ethBalance.Cmp(big.NewInt(0)) < 0 {
			fmt.Printf("got negative balance for ethBalance for networkID %d", networkID)
		}

		if ethBalance.Cmp(big.NewInt(0)) > 0 {
			success, _, _ := services.EthOpsWrapper.TransferToken(services.EthWrappers[networkID],
				services.StringToAddress(account.EthAddress),
				privateKey,
				services.EthWrappers[networkID].MainWalletAddress,
				*tokenBalance,
				services.EthWrappers[networkID].SlowGasPrice)
			if success {
				SetAccountsToNextPaymentStatus([]Account{account})
				return nil
			}
		}
	}

	return errors.New("payment collection failed")
}

/*handleAccountWithCollectionInProgress will check the token balance of an account's payment address. If the balance
is zero, it means the collection has succeeded and the payment status is set to the next status*/
func handleAccountWithCollectionInProgress(account Account) error {
	balanceChecks := []bool{}
	for networkID := range services.EthWrappers {
		balance := services.EthOpsWrapper.GetTokenBalance(services.EthWrappers[networkID],
			services.StringToAddress(account.EthAddress))

		if balance.Cmp(big.NewInt(0)) > 0 {
			balanceChecks = append(balanceChecks, true)
		}
	}

	if len(balanceChecks) == 0 {
		SetAccountsToNextPaymentStatus([]Account{account})
	}

	return nil
}

/*handleAccountAlreadyCollected does not do anything important and is here to prevent index out of range errors*/
func handleAccountAlreadyCollected(account Account) error {
	return nil
}

func GetAllExpiredAccounts(expiredTime time.Time) ([]Account, error) {
	accounts := []Account{}
	if err := DB.Where("expired_at < ?", expiredTime).Find(&accounts).Error; err != nil {
		utils.LogIfError(err, nil)
		return nil, err
	}
	return accounts, nil
}

func DeleteExpiredAccounts(expiredTime time.Time) error {
	accounts, err := GetAllExpiredAccounts(expiredTime)
	if err != nil {
		utils.LogIfError(err, nil)
		return err
	}

	for _, account := range accounts {
		s := ExpiredAccount{
			AccountID:  account.AccountID,
			ExpiredAt:  account.ExpiredAt,
			EthAddress: account.EthAddress,
			RemovedAt:  time.Now(),
		}
		err := DB.Create(&s).Error
		utils.LogIfError(err, nil)
		err = DB.Delete(&account).Error
		utils.LogIfError(err, nil)
	}
	return err
}

/*SetAccountsToLowerPaymentStatusByUpdateTime sets accounts to a lower payment status if the account has a certain payment
status and the updated_at time is older than the cutoff argument*/
func SetAccountsToLowerPaymentStatusByUpdateTime(paymentStatus PaymentStatusType, updatedAtCutoffTime time.Time) error {
	return DB.Exec("UPDATE accounts set payment_status = ? WHERE payment_status = ? AND updated_at < ?", paymentStatus-1, paymentStatus, updatedAtCutoffTime).Error
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

	fmt.Print("ExpiredAt:                      ")
	fmt.Println(account.ExpiredAt)

	fmt.Print("ExpirationDate:                 ")
	fmt.Println(account.ExpirationDate())

	fmt.Print("MonthsInSubscription:           ")
	fmt.Println(account.MonthsInSubscription)

	fmt.Print("Cost:                           ")
	fmt.Println(account.Cost())

	fmt.Print("StorageUsedInByte:                    ")
	fmt.Println(account.StorageUsedInByte)

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

// Temporary func @TODO: remove after migration
func MigrateAccountsToPlanId(planId uint, storageLimit int) {
	query := DB.Exec("UPDATE accounts SET plan_info_id = ? WHERE storage_limit = ?", planId, storageLimit)
	err := query.Error
	utils.LogIfError(err, map[string]interface{}{
		"plan_id":      planId,
		"storageLimit": storageLimit,
	})
}
