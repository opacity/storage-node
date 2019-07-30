package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"math/big"
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

	if err := verifyValidStorageLimit(request.getUpgradeAccountInvoiceObject.StorageLimit, c); err != nil {
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

	if err := verifyValidStorageLimit(request.checkUpgradeStatusObject.StorageLimit, c); err != nil {
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
		if paid {
			amount, err := checkChargeAmount(c, stripePayment.ChargeID)
			if err != nil {
				return InternalErrorResponse(c, err)
			}
			if amount >= upgradeCostInUSD {
				if err := account.UpgradeAccount(request.checkUpgradeStatusObject.StorageLimit,
					request.checkUpgradeStatusObject.DurationInMonths); err != nil {
					return InternalErrorResponse(c, err)
				}
				go func() {
					stripePayment.CheckOPQTransaction()
				}()
				return OkResponse(c, StatusRes{
					Status: "Success with Stripe",
				})
			}
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
	if err := account.UpgradeAccount(request.checkUpgradeStatusObject.StorageLimit,
		request.checkUpgradeStatusObject.DurationInMonths); err != nil {
		return InternalErrorResponse(c, err)
	}
	if err := models.DB.Model(&account).Update("payment_status", models.InitialPaymentReceived).Error; err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, StatusRes{
		Status: "Success with OPQ",
	})
}
