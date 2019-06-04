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

type accountCreateObj struct {
	StorageLimit     int    `json:"storageLimit" binding:"required,gte=100" minimum:"100" maximum:"100" example:"100"`
	DurationInMonths int    `json:"durationInMonths" binding:"required,gte=1" minimum:"1" example:"12"`
	MetadataKey      string `json:"metadataKey" binding:"required,len=64" minLength:"64" maxLength:"64" example:"a 64-char hex string created deterministically from your account handle or private key"`
}

type accountCreateReq struct {
	verification
	requestBody
	accountCreateObj accountCreateObj
}

type accountCreateRes struct {
	ExpirationDate time.Time      `json:"expirationDate" binding:"required,gte"`
	Invoice        models.Invoice `json:"invoice"`
}

type accountDataRes struct {
	PaymentStatus string        `json:"paymentStatus" example:"paid"`
	Error         error         `json:"error" example:"the error encountered while checking"`
	Account       accountGetObj `json:"account" binding:"required"`
}

type accountUnpaidRes struct {
	accountDataRes
	Invoice models.Invoice `json:"invoice"`
}

type accountGetObj struct {
	CreatedAt            time.Time               `json:"createdAt"`
	UpdatedAt            time.Time               `json:"updatedAt"`
	ExpirationDate       time.Time               `json:"expirationDate" binding:"required"`
	MonthsInSubscription int                     `json:"monthsInSubscription" binding:"required,gte=1" example:"12"`                                                        // number of months in their subscription
	StorageLimit         models.StorageLimitType `json:"storageLimit" binding:"required,gte=100" example:"100"`                                                             // how much storage they are allowed, in GB
	StorageUsed          float64                 `json:"storageUsed" binding:"exists" example:"30"`                                                                         // how much storage they have used, in GB
	EthAddress           string                  `json:"ethAddress" binding:"required,len=42" minLength:"42" maxLength:"42" example:"a 42-char eth address with 0x prefix"` // the eth address they will send payment to
	Cost                 float64                 `json:"cost" binding:"required,gte=0" example:"2.00"`
	ApiVersion           int                     `json:"apiVersion" binding:"required,gte=1"`
}

type accountGetReqObj struct {
	Timestamp int64 `json:"timestamp" binding:"required"`
}

type getAccountDataReq struct {
	verification
	requestBody
	accountGetReqObj accountGetReqObj
}

func (v *accountCreateReq) getObjectRef() interface{} {
	return &v.accountCreateObj
}

func (v *getAccountDataReq) getObjectRef() interface{} {
	return &v.accountGetReqObj
}

// CreateAccountHandler godoc
// @Summary create an account
// @Description create an account
// @Accept  json
// @Produce  json
// @Param accountCreateReq body routes.accountCreateReq true "account creation object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"storageLimit": 100,
// @description 	"durationInMonths": 12,
// @description 	"metadataKey": "a 64-char hex string created deterministically from your account handle or private key",
// @description }
// @Success 200 {object} routes.accountCreateRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 503 {string} string "error encrypting private key: (with the error)"
// @Router /api/v1/accounts [post]
/*CreateAccountHandler is a handler for post requests to create accounts*/
func CreateAccountHandler() gin.HandlerFunc {
	return ginHandlerFunc(createAccount)
}

// CheckAccountPaymentStatusHandler godoc
// @Summary check the payment status of an account
// @Description check the payment status of an account
// @Accept  json
// @Produce  json
// @Param getAccountDataReq body routes.getAccountDataReq true "account payment status check object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"timestamp": 1557346389
// @description }
// @Success 200 {object} routes.accountDataRes
// @Success 200 {object} routes.accountUnpaidRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 404 {string} string "no account with that id: (with your accountID)"
// @Router /api/v1/account-data [post]
/*CheckAccountPaymentStatusHandler is a handler for requests checking the payment status*/
func CheckAccountPaymentStatusHandler() gin.HandlerFunc {
	return ginHandlerFunc(checkAccountPaymentStatus)
}

func createAccount(c *gin.Context) error {
	request := accountCreateReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}
	
	ethAddr, privKey, err := services.EthWrapper.GenerateWallet()
	if err != nil {
		err = fmt.Errorf("error generating account wallet:  %v", err)
		return BadRequestResponse(c, err)
	}

	storageLimit, ok := models.StorageLimitMap[request.accountCreateObj.StorageLimit]
	if !ok {
		return BadRequestResponse(c, errors.New("storage not offered in that increment in GB"))
	}

	accountId, err := request.getAccountId(c)
	if err != nil {
		return err
	}

	encryptedKeyInBytes, encryptErr := utils.EncryptWithErrorReturn(
		utils.Env.EncryptionKey,
		privKey,
		accountId,
	)

	if encryptErr != nil {
		return ServiceUnavailableResponse(c, fmt.Errorf("error encrypting private key:  %v", encryptErr))
	}

	account := models.Account{
		AccountID:            accountId,
		MetadataKey:          request.accountCreateObj.MetadataKey,
		StorageLimit:         storageLimit,
		EthAddress:           ethAddr.String(),
		EthPrivateKey:        hex.EncodeToString(encryptedKeyInBytes),
		PaymentStatus:        models.InitialPaymentInProgress,
		MonthsInSubscription: request.accountCreateObj.DurationInMonths,
	}

	// Add account to DB
	if err := models.DB.Create(&account).Error; err != nil {
		return BadRequestResponse(c, err)
	}

	cost, err := account.Cost()
	if err != nil {
		return BadRequestResponse(c, err)
	}

	response := accountCreateRes{
		Invoice: models.Invoice{
			Cost:       cost,
			EthAddress: ethAddr.String(),
		},
		ExpirationDate: account.ExpirationDate(),
	}

	if err := utils.Validator.Struct(&response); err != nil {
		return BadRequestResponse(c, fmt.Errorf("could not create a valid response:  %v", err))
	}

	return OkResponse(c, response)
}

func checkAccountPaymentStatus(c *gin.Context) error {
	request := getAccountDataReq{}
	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	pending := false
	paid, err := account.CheckIfPaid()

	if !paid && err == nil {
		pending, err = account.CheckIfPending()
	}

	cost, _ := account.Cost()
	paymentStatus := createPaymentStatusResponse(paid, pending)
	res := accountDataRes{
		PaymentStatus: createPaymentStatusResponse(paid, pending),
		Error:         err,
		Account: accountGetObj{
			CreatedAt:            account.CreatedAt,
			UpdatedAt:            account.UpdatedAt,
			ExpirationDate:       account.ExpirationDate(),
			MonthsInSubscription: account.MonthsInSubscription,
			StorageLimit:         account.StorageLimit,
			StorageUsed:          float64(account.StorageUsedInByte) / 1e9,
			EthAddress:           account.EthAddress,
			Cost:                 cost,
			ApiVersion:           account.ApiVersion,
		},
	}

	if paymentStatus == Paid {
		return OkResponse(c, res)
	}

	return OkResponse(c, accountUnpaidRes{
		accountDataRes: res,
		Invoice: models.Invoice{
			Cost:       cost,
			EthAddress: account.EthAddress,
		},
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

