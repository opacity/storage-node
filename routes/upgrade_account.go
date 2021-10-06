package routes

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

type getUpgradeAccountInvoiceObject struct {
	PlanId uint `json:"planId" validate:"required,gte=1" minimum:"1" example: "4"`
}

type checkUpgradeStatusObject struct {
	MetadataKeys []string `json:"metadataKeys" validate:"required" example:"an array containing all your metadata keys"`
	FileHandles  []string `json:"fileHandles" validate:"required" example:"an array containing all your file handles"`
	PlanID       uint     `json:"planId" validate:"required,gte=1" minimum:"1" example: "4"`
}

type getUpgradeAccountInvoiceReq struct {
	verification
	requestBody
	getUpgradeAccountInvoiceObject getUpgradeAccountInvoiceObject
}

type checkUpgradeStatusReq struct {
	verification
	requestBody
	checkUpgradeStatusObject checkUpgradeStatusObject
}

type getUpgradeAccountInvoiceRes struct {
	OpctInvoice models.Invoice `json:"opctInvoice"`
	// TODO: uncomment out if we decide to support stripe for upgrades
	// UsdInvoice float64        `json:"usdInvoice,omitempty"`
}

func (v *getUpgradeAccountInvoiceReq) getObjectRef() interface{} {
	return &v.getUpgradeAccountInvoiceObject
}

func (v *checkUpgradeStatusReq) getObjectRef() interface{} {
	return &v.checkUpgradeStatusObject
}

// GetAccountUpgradeInvoiceHandler godoc
// @Summary get an invoice to upgrade an account
// @Description get an invoice to upgrade an account
// @Accept json
// @Produce json
// @Param getUpgradeAccountInvoiceReq body routes.getUpgradeAccountInvoiceReq true "get upgrade invoice object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"planId": 1,
// @description }
// @Success 200 {object} routes.getUpgradeAccountInvoiceRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 404 {string} string "no account with that id: (with your accountID) or plan not found"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/upgrade/invoice [post]
/*GetAccountUpgradeInvoiceHandler is a handler for getting an invoice to upgrade an account*/
func GetAccountUpgradeInvoiceHandler() gin.HandlerFunc {
	return ginHandlerFunc(getAccountUpgradeInvoice)
}

// CheckUpgradeStatusHandler godoc
// @Summary check the upgrade status
// @Description check the upgrade status
// @Accept json
// @Produce json
// @Param checkUpgradeStatusReq body routes.checkUpgradeStatusReq true "check upgrade status object"
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
// @Router /api/v1/upgrade [post]
/*CheckUpgradeStatusHandler is a handler for checking the upgrade status*/
func CheckUpgradeStatusHandler() gin.HandlerFunc {
	return ginHandlerFunc(checkUpgradeStatus)
}

func getAccountUpgradeInvoice(c *gin.Context) error {
	request := getUpgradeAccountInvoiceReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	newPlanInfo, err := models.GetPlanInfoByID(request.getUpgradeAccountInvoiceObject.PlanId)
	if err != nil {
		return NotFoundResponse(c, PlanDoesNotExitErr)
	}

	if err := verifyUpgradeEligible(account, newPlanInfo, c); err != nil {
		return err
	}

	upgradeCostInOPCT, _ := account.UpgradeCostInOPCT(newPlanInfo)

	ethAddr, privKey := services.GenerateWallet()
	encryptedKeyInBytes, encryptErr := utils.EncryptWithErrorReturn(
		utils.Env.EncryptionKey,
		privKey,
		account.AccountID,
	)

	if encryptErr != nil {
		return ServiceUnavailableResponse(c, fmt.Errorf("error encrypting private key:  %v", encryptErr))
	}

	upgrade := models.Upgrade{
		AccountID:     account.AccountID,
		NewPlanInfoID: newPlanInfo.ID,
		OldPlanInfoID: account.PlanInfoID,
		EthAddress:    ethAddr.String(),
		EthPrivateKey: hex.EncodeToString(encryptedKeyInBytes),
		PaymentStatus: models.InitialPaymentInProgress,
		OpctCost:      upgradeCostInOPCT,
		//UsdCost:          upgradeCostInUSD,
		NetworkIdPaid: utils.TestNetworkID,
	}

	upgradeInDB, err := models.GetOrCreateUpgrade(upgrade)
	if err != nil {
		err = fmt.Errorf("error getting or creating upgrade:  %v", err)
		return ServiceUnavailableResponse(c, err)
	}

	return OkResponse(c, getUpgradeAccountInvoiceRes{
		OpctInvoice: models.Invoice{
			Cost:       upgradeCostInOPCT,
			EthAddress: upgradeInDB.EthAddress,
		},
		//UsdInvoice: upgradeCostInUSD,
	})
}

func checkUpgradeStatus(c *gin.Context) error {
	request := checkUpgradeStatusReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	newPlanInfo, err := models.GetPlanInfoByID(request.checkUpgradeStatusObject.PlanID)
	if err != nil {
		return NotFoundResponse(c, PlanDoesNotExitErr)
	}

	if err := verifyUpgradeEligible(account, newPlanInfo, c); err != nil {
		return err
	}

	upgrade, err := models.GetUpgradeFromAccountIDAndPlans(account.AccountID, newPlanInfo.ID, account.PlanInfo.ID)
	//if upgrade.DurationInMonths != request.checkUpgradeStatusObject.DurationInMonths {
	//	return ForbiddenResponse(c, errors.New("durationInMonths does not match durationInMonths "+
	//		"when upgrade was initiated"))
	//}

	//stripePayment, err := models.GetNewestStripePaymentByAccountId(account.AccountID)
	//if stripePayment.AccountID == account.AccountID && err == nil && stripePayment.UpgradePayment {
	//	paid, err := stripePayment.CheckChargePaid()
	//	if err != nil {
	//		return InternalErrorResponse(c, err)
	//	}
	//	if !paid {
	//		return OkResponse(c, StatusRes{
	//			Status: "Incomplete",
	//		})
	//	}
	//	stripePayment.CheckUpgradeOPCTTransaction(account, request.checkUpgradeStatusObject.StorageLimit)
	//	amount, err := checkChargeAmount(c, stripePayment.ChargeID)
	//	if err != nil {
	//		return InternalErrorResponse(c, err)
	//	}
	//	if amount >= upgrade.UsdCost {
	//		if err := upgradeAccountAndUpdateExpireDates(account, request, c); err != nil {
	//			return InternalErrorResponse(c, err)
	//		}
	//		return OkResponse(c, StatusRes{
	//			Status: "Success with Stripe",
	//		})
	//	}
	//}

	paid, networkID, err := models.BackendManager.CheckIfPaid(services.StringToAddress(upgrade.EthAddress),
		services.ConvertToWeiUnit(big.NewFloat(upgrade.OpctCost)))
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	if !paid {
		return OkResponse(c, StatusRes{
			Status: "Incomplete",
		})
	}
	if err := models.DB.Model(&upgrade).Update("payment_status", models.InitialPaymentReceived).Error; err != nil {
		return InternalErrorResponse(c, err)
	}
	if err := upgrade.UpdateNetworkIdPaid(networkID); err != nil {
		return InternalErrorResponse(c, err)
	}
	if err := upgradeAccountAndUpdateExpireDates(account, newPlanInfo, request.checkUpgradeStatusObject.FileHandles, request.checkUpgradeStatusObject.MetadataKeys, request.verification.PublicKey, c); err != nil {
		return InternalErrorResponse(c, err)
	}
	return OkResponse(c, StatusRes{
		Status: "Success with OPCT",
	})
}

func upgradeAccountAndUpdateExpireDates(account models.Account, newPlanInfo utils.PlanInfo, fileHandles []string, metadataKeys []string, publicKey string, c *gin.Context) error {
	if err := account.UpgradeAccount(newPlanInfo); err != nil {
		return err
	}
	filesErr := models.UpdateExpiredAt(fileHandles, publicKey, account.ExpirationDate())

	// Setting ttls on metadata to 2 months post account expiration date so the metadatas won't
	// be deleted too soon
	metadatasErr := updateMetadataExpiration(metadataKeys, publicKey, account.ExpirationDate().Add(MetadataExpirationOffset), c)

	return utils.CollectErrors([]error{filesErr, metadatasErr})
}

func updateMetadataExpiration(metadataKeys []string, key string, newExpiredAtTime time.Time, c *gin.Context) error {
	var kvPairs = make(utils.KVPairs)
	var kvKeys utils.KVKeys

	for _, metadataKey := range metadataKeys {
		permissionHashKey := getPermissionHashKeyForBadger(metadataKey)
		permissionHashValue, _, err := utils.GetValueFromKV(permissionHashKey)
		if err != nil {
			return err
		}

		if err := verifyPermissions(key, metadataKey,
			permissionHashValue, c); err != nil {
			return err
		}
		kvPairs[permissionHashKey] = permissionHashValue
		kvKeys = append(kvKeys, metadataKey)
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
