package routes

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

type getUpgradeV2AccountInvoiceObject struct {
	PlanId uint `json:"planId" validate:"required,gte=1" minimum:"1" example: "4"`
}

type checkUpgradeV2StatusObject struct {
	MetadataKeys []string `json:"metadataKeys" validate:"required" example:"an array containing all your metadata keys"`
	FileHandles  []string `json:"fileHandles" validate:"required" example:"an array containing all your file handles"`
	PlanId       uint     `json:"planId" validate:"required,gte=1" minimum:"1" example: "4"`
}

type getUpgradeV2AccountInvoiceReq struct {
	verification
	requestBody
	getUpgradeV2AccountInvoiceObject getUpgradeV2AccountInvoiceObject
}

type checkUpgradeV2StatusReq struct {
	verification
	requestBody
	checkUpgradeV2StatusObject checkUpgradeV2StatusObject
}

type getUpgradeV2AccountInvoiceRes struct {
	OpctInvoice models.Invoice `json:"opctInvoice"`
	// TODO: uncomment out if we decide to support stripe for upgradeV2s
	// UsdInvoice float64        `json:"usdInvoice,omitempty"`
}

func (v *getUpgradeV2AccountInvoiceReq) getObjectRef() interface{} {
	return &v.getUpgradeV2AccountInvoiceObject
}

func (v *checkUpgradeV2StatusReq) getObjectRef() interface{} {
	return &v.checkUpgradeV2StatusObject
}

// GetAccountUpgradeV2InvoiceHandler godoc
// @Summary get an invoice to upgradeV2 an account
// @Description get an invoice to upgradeV2 an account
// @Accept json
// @Produce json
// @Param getUpgradeV2AccountInvoiceReq body routes.getUpgradeV2AccountInvoiceReq true "get upgradeV2 invoice object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"planId": 1,
// @description }
// @Success 200 {object} routes.getUpgradeV2AccountInvoiceRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 404 {string} string "no account with that id: (with your accountID) or plan not found"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/upgradeV2/invoice [post]
/*GetAccountUpgradeV2InvoiceHandler is a handler for getting an invoice to upgradeV2 an account*/
func GetAccountUpgradeV2InvoiceHandler() gin.HandlerFunc {
	return ginHandlerFunc(getAccountUpgradeV2Invoice)
}

// CheckUpgradeV2StatusHandler godoc
// @Summary check the upgradeV2 status
// @Description check the upgradeV2 status
// @Accept json
// @Produce json
// @Param checkUpgradeV2StatusReq body routes.checkUpgradeV2StatusReq true "check upgradeV2 status object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"planId": 1,
// @description 	"metadataKeys": "["someKey", "someOtherKey]",
// @description 	"fileHandles": "["someHandle", "someOtherHandle]",
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 404 {string} string "no account with that id: (with your accountID) or plan not found"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v2/upgradeV2 [post]
/*CheckUpgradeV2StatusHandler is a handler for checking the upgradeV2 status*/
func CheckUpgradeV2StatusHandler() gin.HandlerFunc {
	return ginHandlerFunc(checkUpgradeV2Status)
}

func getAccountUpgradeV2Invoice(c *gin.Context) error {
	request := getUpgradeV2AccountInvoiceReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	newPlanInfo, err := models.GetPlanInfoByID(request.getUpgradeV2AccountInvoiceObject.PlanId)
	if err != nil {
		return NotFoundResponse(c, PlanDoesNotExitErr)
	}

	if err := verifyUpgradeEligible(account, newPlanInfo, c); err != nil {
		return err
	}

	upgradeV2CostInOPCT, _ := account.UpgradeCostInOPCT(newPlanInfo)

	ethAddr, privKey := services.GenerateWallet()

	encryptedKeyInBytes, encryptErr := utils.EncryptWithErrorReturn(
		utils.Env.EncryptionKey,
		privKey,
		account.AccountID,
	)

	if encryptErr != nil {
		return ServiceUnavailableResponse(c, fmt.Errorf("error encrypting private key:  %v", encryptErr))
	}

	upgradeV2 := models.Upgrade{
		AccountID:     account.AccountID,
		NewPlanInfoID: newPlanInfo.ID,
		OldPlanInfoID: account.PlanInfoID,
		EthAddress:    ethAddr.String(),
		EthPrivateKey: hex.EncodeToString(encryptedKeyInBytes),
		PaymentStatus: models.InitialPaymentInProgress,
		OpctCost:      upgradeV2CostInOPCT,
		//UsdCost:          upgradeCostInUSD,
		NetworkIdPaid: utils.TestNetworkID,
	}

	upgradeV2InDB, err := models.GetOrCreateUpgrade(upgradeV2)
	if err != nil {
		err = fmt.Errorf("error getting or creating upgradeV2:  %v", err)
		return ServiceUnavailableResponse(c, err)
	}

	return OkResponse(c, getUpgradeV2AccountInvoiceRes{
		OpctInvoice: models.Invoice{
			Cost:       upgradeV2CostInOPCT,
			EthAddress: upgradeV2InDB.EthAddress,
		},
		//UsdInvoice: upgradeV2CostInUSD,
	})
}

func checkUpgradeV2Status(c *gin.Context) error {
	request := checkUpgradeV2StatusReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	newPlanInfo, err := models.GetPlanInfoByID(request.checkUpgradeV2StatusObject.PlanId)
	if err != nil {
		return NotFoundResponse(c, PlanDoesNotExitErr)
	}

	if err := verifyUpgradeEligible(account, newPlanInfo, c); err != nil {
		return err
	}

	upgradeV2, err := models.GetUpgradeFromAccountIDAndPlans(account.AccountID, newPlanInfo.ID, account.PlanInfo.ID)
	//if upgradeV2.DurationInMonths != request.checkUpgradeV2StatusObject.DurationInMonths {
	//	return ForbiddenResponse(c, errors.New("durationInMonths does not match durationInMonths "+
	//		"when upgradeV2 was initiated"))
	//}

	//stripePayment, err := models.GetNewestStripePaymentByAccountId(account.AccountID)
	//if stripePayment.AccountID == account.AccountID && err == nil && stripePayment.UpgradeV2Payment {
	//	paid, err := stripePayment.CheckChargePaid()
	//	if err != nil {
	//		return InternalErrorResponse(c, err)
	//	}
	//	if !paid {
	//		return OkResponse(c, StatusRes{
	//			Status: "Incomplete",
	//		})
	//	}
	//	stripePayment.CheckUpgradeV2OPCTTransaction(account, request.checkUpgradeV2StatusObject.StorageLimit)
	//	amount, err := checkChargeAmount(c, stripePayment.ChargeID)
	//	if err != nil {
	//		return InternalErrorResponse(c, err)
	//	}
	//	if amount >= upgradeV2.UsdCost {
	//		if err := upgradeV2AccountAndUpdateExpireDates(account, request, c); err != nil {
	//			return InternalErrorResponse(c, err)
	//		}
	//		return OkResponse(c, StatusRes{
	//			Status: "Success with Stripe",
	//		})
	//	}
	//}

	paid, networkID, err := models.BackendManager.CheckIfPaid(services.StringToAddress(upgradeV2.EthAddress),
		services.ConvertToWeiUnit(big.NewFloat(upgradeV2.OpctCost)))
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	if !paid {
		return OkResponse(c, StatusRes{
			Status: "Incomplete",
		})
	}
	if err := models.DB.Model(&upgradeV2).Update("payment_status", models.InitialPaymentReceived).Error; err != nil {
		return InternalErrorResponse(c, err)
	}
	if err := upgradeV2.UpdateNetworkIdPaid(networkID); err != nil {
		return InternalErrorResponse(c, err)
	}
	if err := upgradeV2AccountAndUpdateExpireDates(account, newPlanInfo, request.checkUpgradeV2StatusObject.FileHandles, request.checkUpgradeV2StatusObject.MetadataKeys, request.verification.PublicKey, c); err != nil {
		return InternalErrorResponse(c, err)
	}
	return OkResponse(c, StatusRes{
		Status: "Success with OPCT",
	})
}

func upgradeV2AccountAndUpdateExpireDates(account models.Account, newPlanInfo utils.PlanInfo, fileHandles []string, metadataKeys []string, publicKey string, c *gin.Context) error {
	if err := account.UpgradeAccount(newPlanInfo); err != nil {
		return err
	}
	filesErr := models.UpdateExpiredAt(fileHandles, publicKey, account.ExpirationDate())

	// Setting ttls on metadata to 2 months post account expiration date so the metadatas won't
	// be deleted too soon
	metadatasErr := updateMetadataExpiration(metadataKeys, publicKey, account.ExpirationDate().Add(MetadataExpirationOffset), c)

	return utils.CollectErrors([]error{filesErr, metadatasErr})
}

func updateMetadataExpirationV2(metadataKeys []string, key string, newExpiredAtTime time.Time, c *gin.Context) error {
	var kvPairs = make(utils.KVPairs)
	var kvKeys utils.KVKeys

	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		return err
	}

	for _, metadataKey := range metadataKeys {
		metadataKeyBytes, err := base64.RawURLEncoding.DecodeString(metadataKey)
		if err != nil {
			return err
		}

		permissionHashKey := getPermissionHashKeyForBadger(string(metadataKeyBytes))
		permissionHashValue, _, err := utils.GetValueFromKV(permissionHashKey)
		if err != nil {
			return err
		}

		if err := verifyPermissionsV2(keyBytes, metadataKeyBytes,
			permissionHashValue, c); err != nil {
			return err
		}
		kvPairs[permissionHashKey] = permissionHashValue
		kvKeys = append(kvKeys, string(metadataKeyBytes))
	}

	kvs, err := utils.BatchGet(&kvKeys)
	if err != nil {
		return err
	}
	for key, value := range *kvs {
		kvPairs[key] = value
	}

	if err := utils.BatchSet(&kvPairs, time.Until(newExpiredAtTime)); err != nil {
		return err
	}

	return nil
}
