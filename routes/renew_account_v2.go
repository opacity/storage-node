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

type getRenewalV2AccountInvoiceObject struct {
}

type checkRenewalV2StatusObject struct {
	MetadataKeys []string `json:"metadataKeys" validate:"required" example:"an array containing all your metadata keys"`
	FileHandles  []string `json:"fileHandles" validate:"required" example:"an array containing all your file handles"`
	NetworkID    uint     `json:"networkId" validate:"required,gte=0" example:"1"`
}

type getRenewalV2AccountInvoiceReq struct {
	verification
	requestBody
	getRenewalV2AccountInvoiceObject getRenewalV2AccountInvoiceObject
}

type checkRenewalV2StatusReq struct {
	verification
	requestBody
	checkRenewalV2StatusObject checkRenewalV2StatusObject
}

type getRenewalV2AccountInvoiceRes struct {
	OpctInvoice models.Invoice `json:"opctInvoice"`
	// TODO: uncomment out if we decide to support stripe for renewalV2s
	// UsdInvoice float64        `json:"usdInvoice"`
}

func (v *getRenewalV2AccountInvoiceReq) getObjectRef() interface{} {
	return &v.getRenewalV2AccountInvoiceObject
}

func (v *checkRenewalV2StatusReq) getObjectRef() interface{} {
	return &v.checkRenewalV2StatusObject
}

// GetAccountRenewalV2InvoiceHandler godoc
// @Summary get an invoice to renew an account
// @Description get an invoice to renew an account
// @Accept  json
// @Produce  json
// @Param getRenewalV2AccountInvoiceReq body routes.getRenewalV2AccountInvoiceReq true "get renewalV2 invoice object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description }
// @Success 200 {object} routes.getRenewalV2AccountInvoiceRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 404 {string} string "no account with that id: (with your accountID)"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v2/renew/invoice [post]
/*GetAccountRenewalV2InvoiceHandler is a handler for getting an invoice to renew an account*/
func GetAccountRenewalV2InvoiceHandler() gin.HandlerFunc {
	return ginHandlerFunc(getAccountRenewalV2Invoice)
}

// CheckRenewalV2StatusHandler godoc
// @Summary check the renewalV2 status
// @Description check the renewalV2 status
// @Accept  json
// @Produce  json
// @Param checkRenewalV2StatusReq body routes.checkRenewalV2StatusReq true "check renewalV2 status object"
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
// @Router /api/v2/renew [post]
/*CheckRenewalV2StatusHandler is a handler for checking the renewalV2 status*/
func CheckRenewalV2StatusHandler() gin.HandlerFunc {
	return ginHandlerFunc(checkRenewalV2Status)
}

func getAccountRenewalV2Invoice(c *gin.Context) error {
	request := getRenewalV2AccountInvoiceReq{}

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

	renewalV2CostInOPCT, err := account.Cost()
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	//renewalV2CostInUSD := utils.Env.Plans[int(account.StorageLimit)].CostInUSD

	ethAddr, privKey := services.GenerateWallet()

	encryptedKeyInBytes, encryptErr := utils.EncryptWithErrorReturn(
		utils.Env.EncryptionKey,
		privKey,
		account.AccountID,
	)

	if encryptErr != nil {
		return ServiceUnavailableResponse(c, fmt.Errorf("error encrypting private key:  %v", encryptErr))
	}

	renewalV2 := models.Renewal{
		AccountID:        account.AccountID,
		EthAddress:       ethAddr.String(),
		EthPrivateKey:    hex.EncodeToString(encryptedKeyInBytes),
		PaymentStatus:    models.InitialPaymentInProgress,
		OpctCost:         renewalV2CostInOPCT,
		DurationInMonths: 12,
	}

	renewalV2InDB, err := models.GetOrCreateRenewal(renewalV2)
	if err != nil {
		err = fmt.Errorf("error getting or creating renewalV2:  %v", err)
		return ServiceUnavailableResponse(c, err)
	}

	return OkResponse(c, getRenewalV2AccountInvoiceRes{
		OpctInvoice: models.Invoice{
			Cost:       renewalV2CostInOPCT,
			EthAddress: renewalV2InDB.EthAddress,
		},
		// TODO: uncomment out if we decide to support stripe for renewalV2s
		// UsdInvoice: renewalV2CostInUSD,
	})
}

func checkRenewalV2Status(c *gin.Context) error {
	request := checkRenewalV2StatusReq{}

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

	renewalV2s, err := models.GetRenewalsFromAccountID(account.AccountID)
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	if len(renewalV2s) == 0 {
		return NotFoundResponse(c, errors.New("no renewalV2s found"))
	}

	paid, err := models.BackendManager.CheckIfPaid(services.StringToAddress(renewalV2s[0].EthAddress),
		services.ConvertToWeiUnit(big.NewFloat(renewalV2s[0].OpctCost)), request.checkRenewalV2StatusObject.NetworkID)
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	if !paid {
		return OkResponse(c, StatusRes{
			Status: "Incomplete",
		})
	}

	if renewalV2s[0].PaymentStatus >= models.InitialPaymentReceived {
		return OkResponse(c, StatusRes{
			Status: "Success with OPCT",
		})
	}

	if err := models.DB.Model(&renewalV2s[0]).Update("payment_status", models.InitialPaymentReceived).Error; err != nil {
		return InternalErrorResponse(c, err)
	}
	if err := renewalV2AccountAndUpdateExpireDates(account, request, c); err != nil {
		return InternalErrorResponse(c, err)
	}
	return OkResponse(c, StatusRes{
		Status: "Success with OPCT",
	})
}

func renewalV2AccountAndUpdateExpireDates(account models.Account, request checkRenewalV2StatusReq, c *gin.Context) error {
	if err := account.RenewAccount(); err != nil {
		return err
	}
	filesErr := models.UpdateExpiredAt(request.checkRenewalV2StatusObject.FileHandles,
		request.verification.PublicKey, account.ExpirationDate())

	// Setting ttls on metadata to 2 months post account expiration date so the metadatas won't
	// be deleted too soon
	metadatasErr := updateMetadataExpirationV2(request.checkRenewalV2StatusObject.MetadataKeys,
		request.verification.PublicKey, account.ExpirationDate().Add(MetadataExpirationOffset), c)

	return utils.CollectErrors([]error{filesErr, metadatasErr})
}
