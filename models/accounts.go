package models

import (
	"errors"
	"fmt"
	"time"

	"math/big"

	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

/*Account defines a model for managing a user subscription for uploads*/
type Account struct {
	AccountID            string            `gorm:"primary_key" json:"accountID" binding:"required,len=64"` // some hash of the user's master handle
	CreatedAt            time.Time         `json:"createdAt"`
	UpdatedAt            time.Time         `json:"updatedAt"`
	MonthsInSubscription int               `json:"monthsInSubscription" binding:"required,gte=1"` // number of months in their subscription
	StorageLocation      string            `json:"storageLocation" binding:"required,url"`        // where their files live, on S3 or elsewhere
	StorageLimit         StorageLimitType  `json:"storageLimit" binding:"required,gte=100"`       // how much storage they are allowed, in GB
	StorageUsed          float64           `json:"storageUsed" binding:"required"`                // how much storage they have used, in GB
	EthAddress           string            `json:"ethAddress" binding:"required,len=42"`          // the eth address they will send payment to
	EthPrivateKey        string            `json:"ethPrivateKey" binding:"required,len=96"`       // the private key of the eth address
	PaymentStatus        PaymentStatusType `json:"paymentStatus" binding:"required"`              // the status of their payment
}

/*Invoice is the invoice object we will return to the client*/
type Invoice struct {
	Cost       float64 `json:"cost" binding:"required,gte=0"`
	EthAddress string  `json:"ethAddress" binding:"required,len=42"`
}

/*StorageLimitType defines a type for the storage limits*/
type StorageLimitType int

/*PaymentStatusType defines a type for the payment statuses*/
type PaymentStatusType int

const (
	/*BasicStorageLimit allows 100 GB on the basic plan*/
	BasicStorageLimit StorageLimitType = iota + 100
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
const BasicSubscriptionDefaultCost = 1.56

/*PaymentStatusMap is for pretty printing the PaymentStatus*/
var PaymentStatusMap = make(map[PaymentStatusType]string)

/*StorageLimitMap maps the amount of storage passed in by the client
to storage limits set by us.  Used for checking that they have passed in an allowed
amount.*/
var StorageLimitMap = make(map[int]StorageLimitType)

/*CostMap is for mapping subscription plans to their prices, for a default length subscription*/
var CostMap = make(map[StorageLimitType]float64)

func init() {
	StorageLimitMap[int(BasicStorageLimit)] = BasicStorageLimit

	PaymentStatusMap[InitialPaymentInProgress] = "InitialPaymentInProgress"
	PaymentStatusMap[InitialPaymentReceived] = "InitialPaymentReceived"
	PaymentStatusMap[GasTransferInProgress] = "GasTransferInProgress"
	PaymentStatusMap[GasTransferComplete] = "GasTransferComplete"
	PaymentStatusMap[PaymentRetrievalInProgress] = "PaymentRetrievalInProgress"
	PaymentStatusMap[PaymentRetrievalComplete] = "PaymentRetrievalComplete"

	CostMap[BasicStorageLimit] = BasicSubscriptionDefaultCost
}

/*BeforeCreate - callback called before the row is created*/
func (account *Account) BeforeCreate(scope *gorm.Scope) error {
	if account.PaymentStatus < InitialPaymentInProgress {
		account.PaymentStatus = InitialPaymentInProgress
	}
	return nil
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
		account.PaymentStatus = InitialPaymentReceived
		DB.Update(&account)
	}
	return paid, err
}

func (account *Account) UseStorageSpaceInByte(planToUsedInByte int) error {
	paid, err := account.CheckIfPaid()
	if err != nil {
		return err
	}
	if !paid {
		return errors.New("Not payment. Unable to update the storage")
	}

	inGb := float64(planToUsedInByte) / float64(1e9)
	if inGb+account.StorageUsed > float64(account.StorageLimit) {
		return errors.New("Unable to store more data")
	}
	account.StorageUsed = account.StorageUsed + inGb
	return DB.Update(*account).Error
}

/*Return Account object(first one) if there is not any error. */
func GetAccountById(accountID string) (Account, error) {
	account := Account{}
	err := DB.Where("account_id = ?", accountID).First(&account).Error
	return account, err
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

	fmt.Print("StorageUsed:                   ")
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
