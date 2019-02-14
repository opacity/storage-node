package models

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
)

/*Account defines a model for managing a user subscription for uploads*/
type Account struct {
	UserID          string            `gorm:"primary_key" json:"user_id" binding:"required,len=64"` // some hash of the user's master handle
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
	ExpirationDate  time.Time         `json:"expirationDate" binding:"required,gte"`   // when their subscription expires
	StorageLocation string            `json:"storageLocation" binding:"required,url"`  // where their files live, on S3 or elsewhere
	StorageLimit    StorageLimitType  `json:"storageLimit" binding:"required,gte=100"` // how much storage they are allowed, in GB
	PIN             int               `json:"pin" binding:"required"`                  // the user's PIN (do we need this field?)
	EthAddress      string            `json:"ethAddress" binding:"required,len=42"`    // the eth address they will send payment to
	EthPrivateKey   string            `json:"ethPrivateKey" binding:"required,len=96"` // the private key of the eth address
	PaymentStatus   PaymentStatusType `json:"paymentStatus" binding:"required"`        // the status of their payment
}

/*StorageLimitType defines a type for the storage limits*/
type StorageLimitType int

/*PaymentStatusType defines a type for the payment statuses*/
type PaymentStatusType int

const (
	/*BasicStorageLimit allows 100 GB on the basic plan*/
	BasicStorageLimit StorageLimitType = 100
)

const (
	/*InitialPaymentInProgress - we have created the subscription and are awaiting their initial payment*/
	InitialPaymentInProgress PaymentStatusType = 1

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
)

/*PaymentStatusMap is for pretty printing the PaymentStatus*/
var PaymentStatusMap = make(map[PaymentStatusType]string)

func init() {
	PaymentStatusMap[InitialPaymentInProgress] = "InitialPaymentInProgress"
	PaymentStatusMap[InitialPaymentReceived] = "InitialPaymentReceived"
	PaymentStatusMap[GasTransferInProgress] = "GasTransferInProgress"
	PaymentStatusMap[GasTransferComplete] = "GasTransferComplete"
	PaymentStatusMap[PaymentRetrievalInProgress] = "PaymentRetrievalInProgress"
	PaymentStatusMap[PaymentRetrievalComplete] = "PaymentRetrievalComplete"
}

/*BeforeCreate - callback called before the row is created*/
func (account *Account) BeforeCreate(scope *gorm.Scope) error {
	if account.PaymentStatus < InitialPaymentInProgress {
		account.PaymentStatus = InitialPaymentInProgress
	}
	return nil
}

/*PrettyString - print the account in a friendly way.  Not used for external logging, just for watching in the
terminal*/
func (account *Account) PrettyString() {
	fmt.Print("UserID:                         ")
	fmt.Println(account.UserID)

	fmt.Print("CreatedAt:                      ")
	fmt.Println(account.CreatedAt)

	fmt.Print("UpdatedAt:                      ")
	fmt.Println(account.UpdatedAt)

	fmt.Print("ExpirationDate:                 ")
	fmt.Println(account.ExpirationDate)

	fmt.Print("StorageLimit:                   ")
	fmt.Println(account.StorageLimit)

	fmt.Print("StorageLocation:                ")
	fmt.Println(account.StorageLocation)

	fmt.Print("PaymentStatus:                  ")
	fmt.Println(PaymentStatusMap[account.PaymentStatus])

	fmt.Print("EthAddress:                     ")
	fmt.Println(account.EthAddress)

	fmt.Print("EthPrivateKey:                  ")
	fmt.Println(account.EthPrivateKey)

	fmt.Print("PIN:                            ")
	fmt.Println(account.PIN)
}
