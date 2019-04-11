package routes

import (
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

const Unpaid = "unpaid"
const Pending = "pending"
const Paid = "paid"

type accountCreateReq struct {
	AccountID        string `json:"accountID" binding:"required,len=64"`
	StorageLimit     int    `json:"storageLimit" binding:"required,gte=100"`
	DurationInMonths int    `json:"durationInMonths" binding:"required,gte=1"`
	MetadataKey      string `json:"metadataKey" binding:"required,len=64"`
}

type accountCreateRes struct {
	ExpirationDate time.Time      `json:"expirationDate" binding:"required,gte"`
	Invoice        models.Invoice `json:"invoice"`
}

type accountPaidRes struct {
	PaymentStatus string `json:"paymentStatus"`
	Error         error  `json:"error"`
}

/*CreateAccountHandler is a handler for post requests to create accounts*/
func CreateAccountHandler() gin.HandlerFunc {
	return ginHandlerFunc(createAccount)
}

/*CheckAccountPaymentStatusHandler is a handler for requests checking the payment status*/
func CheckAccountPaymentStatusHandler() gin.HandlerFunc {
	return ginHandlerFunc(checkAccountPaymentStatus)
}

func createAccount(c *gin.Context) {
	request := accountCreateReq{}
	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		BadRequestResponse(c, err)
		return
	}

	ethAddr, privKey, err := services.EthWrapper.GenerateWallet()
	if err != nil {
		err = fmt.Errorf("error generating account wallet:  %v", err)
		BadRequestResponse(c, err)
		return
	}

	storageLimit, ok := models.StorageLimitMap[request.StorageLimit]
	if !ok {
		BadRequestResponse(c, errors.New("storage not offered in that increment in GB"))
		return
	}

	encryptedKeyInBytes, encryptErr := utils.EncryptWithErrorReturn(
		utils.Env.EncryptionKey,
		privKey,
		request.AccountID,
	)
	if encryptErr != nil {
		ServiceUnavailableResponse(c, fmt.Errorf("error encrypting private key:  %v", encryptErr))
		return
	}

	account := models.Account{
		AccountID:            request.AccountID,
		MetadataKey:          request.MetadataKey,
		StorageLimit:         storageLimit,
		EthAddress:           ethAddr.String(),
		EthPrivateKey:        hex.EncodeToString(encryptedKeyInBytes),
		PaymentStatus:        models.InitialPaymentInProgress,
		MonthsInSubscription: request.DurationInMonths,
	}

	if err := utils.Validator.Struct(account); err == nil {
		BadRequestResponse(c, err)
		return
	}

	cost, err := account.Cost()
	if err != nil {
		BadRequestResponse(c, err)
		return
	}

	// Add account to DB
	if err := models.DB.Create(&account).Error; err != nil {
		BadRequestResponse(c, err)
		return
	}

	response := accountCreateRes{
		Invoice: models.Invoice{
			Cost:       cost,
			EthAddress: ethAddr.String(),
		},
		ExpirationDate: account.ExpirationDate(),
	}

	if err := utils.Validator.Struct(&response); err != nil {
		err = fmt.Errorf("could not create a valid response:  %v", err)
		BadRequestResponse(c, err)
		return
	}

	utils.Metrics_AccountCreated_Counter.Inc()
	OkResponse(c, response)
}

func checkAccountPaymentStatus(c *gin.Context) {
	accountID := c.Param("accountID")

	account, err := models.GetAccountById(accountID)
	if err != nil {
		AccountNotFoundResponse(c, accountID)
		return
	}

	pending := false
	initialPaymentStatus := account.PaymentStatus
	paid, err := account.CheckIfPaid()

	if paid && err == nil && initialPaymentStatus == models.InitialPaymentInProgress {
		err := models.HandleMetadataKeyForPaidAccount(account)
		if err != nil {
			BadRequestResponse(c, err)
			return
		}
	} else if !paid && err == nil {
		pending, err = account.CheckIfPending()
	}

	OkResponse(c, accountPaidRes{
		PaymentStatus: createPaymentStatusResponse(paid, pending),
		Error:         err,
	})
}

func createPaymentStatusResponse(paid bool, pending bool) string {
	if paid {
		return Paid
	}
	if pending {
		return Pending
	}
	return Unpaid
}
