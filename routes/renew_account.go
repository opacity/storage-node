package routes

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

type getRenewalAccountInvoiceObject struct {
}

type checkRenewalStatusObject struct {
	MetadataKeys []string `json:"metadataKeys" validate:"required" example:"an array containing all your metadata keys"`
	FileHandles  []string `json:"fileHandles" validate:"required" example:"an array containing all your file handles"`
	NetworkID    uint     `json:"networkId" validate:"required,gte=1" example:"1"`
}

type getRenewalAccountInvoiceReq struct {
	verification
	requestBody
	getRenewalAccountInvoiceObject getRenewalAccountInvoiceObject
}

type checkRenewalStatusReq struct {
	verification
	requestBody
	checkRenewalStatusObject checkRenewalStatusObject
}

type getRenewalAccountInvoiceRes struct {
	OpctInvoice models.Invoice `json:"opctInvoice"`
	// TODO: uncomment out if we decide to support stripe for renewals
	// UsdInvoice float64        `json:"usdInvoice"`
}

func (v *getRenewalAccountInvoiceReq) getObjectRef() interface{} {
	return &v.getRenewalAccountInvoiceObject
}

func (v *checkRenewalStatusReq) getObjectRef() interface{} {
	return &v.checkRenewalStatusObject
}

// GetAccountRenewalInvoiceHandler godoc
// @Summary get an invoice to renew an account
// @Description get an invoice to renew an account
// @Accept  json
// @Produce  json
// @Param getRenewalAccountInvoiceReq body routes.getRenewalAccountInvoiceReq true "get renewal invoice object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description }
// @Success 200 {object} routes.getRenewalAccountInvoiceRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 404 {string} string "no account with that id: (with your accountID)"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/renew/invoice [post]
/*GetAccountRenewalInvoiceHandler is a handler for getting an invoice to renew an account*/
func GetAccountRenewalInvoiceHandler() gin.HandlerFunc {
	return ginHandlerFunc(getAccountRenewalInvoice)
}

// CheckRenewalStatusHandler godoc
// @Summary check the renewal status
// @Description check the renewal status
// @Accept json
// @Produce json
// @Param checkRenewalStatusReq body routes.checkRenewalStatusReq true "check renewal status object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"metadataKeys": "["someKey", "someOtherKey]",
// @description 	"fileHandles": "["someHandle", "someOtherHandle]",
// @description   "networkId": 1,
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 404 {string} string "no account with that id: (with your accountID)"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/renew [post]
/*CheckRenewalStatusHandler is a handler for checking the renewal status*/
func CheckRenewalStatusHandler() gin.HandlerFunc {
	return ginHandlerFunc(checkRenewalStatus)
}

func getAccountRenewalInvoice(c *gin.Context) error {
	request := getRenewalAccountInvoiceReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	if err := verifyRenewEligible(account, c); err != nil {
		return err
	}

	renewalCostInOPCT, err := account.Cost()
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	//renewalCostInUSD := utils.Env.Plans[int(account.StorageLimit)].CostInUSD

	ethAddr, privKey := services.GenerateWallet()

	encryptedKeyInBytes, encryptErr := utils.EncryptWithErrorReturn(
		utils.Env.EncryptionKey,
		privKey,
		account.AccountID,
	)

	if encryptErr != nil {
		return ServiceUnavailableResponse(c, fmt.Errorf("error encrypting private key:  %v", encryptErr))
	}

	renewal := models.Renewal{
		AccountID:        account.AccountID,
		EthAddress:       ethAddr.String(),
		EthPrivateKey:    hex.EncodeToString(encryptedKeyInBytes),
		PaymentStatus:    models.InitialPaymentInProgress,
		OpctCost:         renewalCostInOPCT,
		DurationInMonths: 12,
	}

	renewalInDB, err := models.GetOrCreateRenewal(renewal)
	if err != nil {
		err = fmt.Errorf("error getting or creating renewal:  %v", err)
		return ServiceUnavailableResponse(c, err)
	}

	return OkResponse(c, getRenewalAccountInvoiceRes{
		OpctInvoice: models.Invoice{
			Cost:       renewalCostInOPCT,
			EthAddress: renewalInDB.EthAddress,
		},
		// TODO: uncomment out if we decide to support stripe for renewals
		// UsdInvoice: renewalCostInUSD,
	})
}

func checkRenewalStatus(c *gin.Context) error {
	request := checkRenewalStatusReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	if err := verifyRenewEligible(account, c); err != nil {
		return err
	}

	renewals, err := models.GetRenewalsFromAccountID(account.AccountID)
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	if len(renewals) == 0 {
		return NotFoundResponse(c, errors.New("no renewals found"))
	}

	paid, err := models.BackendManager.CheckIfPaid(services.StringToAddress(renewals[0].EthAddress),
		services.ConvertToWeiUnit(big.NewFloat(renewals[0].OpctCost)), request.checkRenewalStatusObject.NetworkID)
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	if !paid {
		return OkResponse(c, StatusRes{
			Status: "Incomplete",
		})
	}

	if renewals[0].PaymentStatus >= models.InitialPaymentReceived {
		return OkResponse(c, StatusRes{
			Status: "Success with OPCT",
		})
	}

	if err := models.DB.Model(&renewals[0]).Update("payment_status", models.InitialPaymentReceived).Error; err != nil {
		return InternalErrorResponse(c, err)
	}
	if err := renewalAccountAndUpdateExpireDates(account, request, c); err != nil {
		return InternalErrorResponse(c, err)
	}
	return OkResponse(c, StatusRes{
		Status: "Success with OPCT",
	})
}

func renewalAccountAndUpdateExpireDates(account models.Account, request checkRenewalStatusReq, c *gin.Context) error {
	if err := account.RenewAccount(); err != nil {
		return err
	}
	filesErr := models.UpdateExpiredAt(request.checkRenewalStatusObject.FileHandles,
		request.verification.PublicKey, account.ExpirationDate())

	// Setting ttls on metadata to 2 months post account expiration date so the metadatas won't
	// be deleted too soon
	metadatasErr := updateMetadataExpiration(request.checkRenewalStatusObject.MetadataKeys,
		request.verification.PublicKey, account.ExpirationDate().Add(MetadataExpirationOffset), c)

	return utils.CollectErrors([]error{filesErr, metadatasErr})
}
