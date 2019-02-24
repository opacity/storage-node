package routes

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

type accountCreateReq struct {
	AccountID        string `json:"accountID" binding:"required,len=64"`
	StorageLimit     int    `json:"storageLimit" binding:"required,gte=100"`
	DurationInMonths int    `json:"durationInMonths" binding:"required,gte=1"`
}

type accountCreateRes struct {
	ExpirationDate time.Time      `json:"expirationDate" binding:"required,gte"`
	Invoice        models.Invoice `json:"invoice"`
}

type accountPaidRes struct {
	Paid  bool  `json:"paid" binding:"required"`
	Error error `json:"error"`
}

const noAccountResponse = "no account with that id"

/*CreateAccountHandler is a handler for post requests to create accounts*/
func CreateAccountHandler() gin.HandlerFunc {
	return gin.HandlerFunc(createAccount)
}

/*CheckAccountPaymentStatusHandler is a handler for requests checking the payment status*/
func CheckAccountPaymentStatusHandler() gin.HandlerFunc {
	return gin.HandlerFunc(checkAccountPaymentStatus)
}

func createAccount(c *gin.Context) {
	request := accountCreateReq{}
	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
		return
	}

	ethAddr, privKey, err := services.EthWrapper.GenerateWallet()
	if err != nil {
		err = fmt.Errorf("error generating account wallet:  %v", err)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, err.Error())
		return
	}

	storageLimit, ok := models.StorageLimitMap[request.StorageLimit]
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, errors.New("storage not offered in that increment in GB"))
		return
	}

	encryptedKeyInBytes, encryptErr := utils.EncryptWithErrorReturn(
		utils.Env.EncryptionKey,
		privKey,
		request.AccountID,
	)
	if encryptErr != nil {
		err = fmt.Errorf("error encrypting private key:  %v", encryptErr)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, err.Error())
		return
	}

	account := models.Account{
		AccountID:            request.AccountID,
		StorageLimit:         storageLimit,
		EthAddress:           ethAddr.String(),
		EthPrivateKey:        hex.EncodeToString(encryptedKeyInBytes),
		PaymentStatus:        models.InitialPaymentInProgress,
		MonthsInSubscription: request.DurationInMonths,
	}

	if err := utils.Validator.Struct(account); err == nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
		return
	}

	cost, err := account.Cost()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
		return
	}

	// Add account to DB
	if err := models.DB.Create(&account).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
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
		c.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, response)
}

func checkAccountPaymentStatus(c *gin.Context) {
	slug := c.Param("accountID")

	account, err := models.GetAccountById(slug)
	if err != nil {
		goto ErrorHandling
	}
	paid, err := account.CheckIfPaid()
	if err != nil {
		goto ErrorHandling
	}
	c.JSON(http.StatusOK, accountPaidRes{
		Paid:  paid,
		Error: err,
	})
	return

ErrorHandling:
	c.JSON(http.StatusNotFound, noAccountResponse)
}
