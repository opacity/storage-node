package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"math/big"
	"time"
)

type getUpgradeAccountInvoiceObject struct {
	StorageLimit     int `json:"storageLimit" binding:"required,gte=128" minimum:"128" example:"128"`
	DurationInMonths int `json:"durationInMonths" binding:"required,gte=1" minimum:"1" example:"12"`
}

type checkUpgradeStatusObject struct {
	MetadataKeys     []string `json:"metadataKeys" binding:"required" example:"an array containing all your metadata keys"`
	FileHandles      []string `json:"fileHandles" binding:"required" example:"an array containing all your file handles"`
	StorageLimit     int      `json:"storageLimit" binding:"required,gte=128" minimum:"128" example:"128"`
	DurationInMonths int      `json:"durationInMonths" binding:"required,gte=1" minimum:"1" example:"12"`
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
	OpqInvoice models.Invoice `json:"opqInvoice"`
	UsdInvoice float64        `json:"usdInvoice"`
}

func (v *getUpgradeAccountInvoiceReq) getObjectRef() interface{} {
	return &v.getUpgradeAccountInvoiceObject
}

func (v *checkUpgradeStatusReq) getObjectRef() interface{} {
	return &v.checkUpgradeStatusObject
}

func GetAccountUpgradeInvoiceHandler() gin.HandlerFunc {
	return ginHandlerFunc(getAccountUpgradeInvoice)
}

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

	if err := verifyUpgradeEligible(int(account.StorageLimit), request.getUpgradeAccountInvoiceObject.StorageLimit, c); err != nil {
		return err
	}

	upgradeCostInOPQ, _ := account.UpgradeCostInOPQ(request.getUpgradeAccountInvoiceObject.StorageLimit,
		request.getUpgradeAccountInvoiceObject.DurationInMonths)
	upgradeCostInUSD, _ := account.UpgradeCostInUSD(request.getUpgradeAccountInvoiceObject.StorageLimit,
		request.getUpgradeAccountInvoiceObject.DurationInMonths)

	return OkResponse(c, getUpgradeAccountInvoiceRes{
		OpqInvoice: models.Invoice{
			Cost:       upgradeCostInOPQ,
			EthAddress: account.EthAddress,
		},
		UsdInvoice: upgradeCostInUSD,
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

	if err := verifyUpgradeEligible(int(account.StorageLimit), request.checkUpgradeStatusObject.StorageLimit, c); err != nil {
		return err
	}

	upgradeCostInOPQ, _ := account.UpgradeCostInOPQ(request.checkUpgradeStatusObject.StorageLimit,
		request.checkUpgradeStatusObject.DurationInMonths)
	upgradeCostInUSD, _ := account.UpgradeCostInUSD(request.checkUpgradeStatusObject.StorageLimit,
		request.checkUpgradeStatusObject.DurationInMonths)

	stripePayment, err := models.GetNewestStripePaymentByAccountId(account.AccountID)
	if stripePayment.AccountID == account.AccountID && err == nil {
		paid, err := stripePayment.CheckChargePaid()
		if err != nil {
			return InternalErrorResponse(c, err)
		}
		if !paid {
			return OkResponse(c, StatusRes{
				Status: "Incomplete",
			})
		}
		stripePayment.CheckOPQTransaction()
		amount, err := checkChargeAmount(c, stripePayment.ChargeID)
		if err != nil {
			return InternalErrorResponse(c, err)
		}
		if amount >= upgradeCostInUSD {
			if err := upgradeAccountAndUpdateExpireDates(account, request, c); err != nil {
				return err
			}
			return OkResponse(c, StatusRes{
				Status: "Success with Stripe",
			})
		}
	}

	paid, err := models.BackendManager.CheckIfPaid(services.StringToAddress(account.EthAddress),
		utils.ConvertToWeiUnit(big.NewFloat(upgradeCostInOPQ)))
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	if !paid {
		return OkResponse(c, StatusRes{
			Status: "Incomplete",
		})
	}
	if err := models.DB.Model(&account).Update("payment_status", models.InitialPaymentReceived).Error; err != nil {
		return InternalErrorResponse(c, err)
	}
	if err := upgradeAccountAndUpdateExpireDates(account, request, c); err != nil {
		return err
	}
	return OkResponse(c, StatusRes{
		Status: "Success with OPQ",
	})
}

func upgradeAccountAndUpdateExpireDates(account models.Account, request checkUpgradeStatusReq, c *gin.Context) error {
	if err := account.UpgradeAccount(request.checkUpgradeStatusObject.StorageLimit,
		request.checkUpgradeStatusObject.DurationInMonths); err != nil {
		return InternalErrorResponse(c, err)
	}
	if err := models.UpdateExpiredAt(request.checkUpgradeStatusObject.FileHandles,
		request.verification.PublicKey, account.ExpirationDate()); err != nil {
		return InternalErrorResponse(c, err)
	}
	if err := updateMetadataExpiration(request.checkUpgradeStatusObject.MetadataKeys,
		request.verification.PublicKey, account.ExpirationDate(), c); err != nil {
		return err
	}
	return nil
}

func updateMetadataExpiration(metadataKeys []string, key string, newExpiredAtTime time.Time, c *gin.Context) error {
	var kvPairs = make(utils.KVPairs)
	var kvKeys utils.KVKeys

	for _, metadataKey := range metadataKeys {
		permissionHashKey := getPermissionHashKeyForBadger(metadataKey)
		permissionHashValue, _, err := utils.GetValueFromKV(permissionHashKey)
		if err != nil {
			return InternalErrorResponse(c, err)
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
		return InternalErrorResponse(c, err)
	}
	for key, value := range *kvs {
		kvPairs[key] = value
	}

	if err := utils.BatchSet(&kvPairs, time.Until(newExpiredAtTime)); err != nil {
		return InternalErrorResponse(c, err)
	}

	return nil
}
