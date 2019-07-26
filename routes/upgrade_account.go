package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
)

type getUpgradeAccountInvoiceObject struct {
	StorageLimit     int `json:"storageLimit" binding:"required,gte=100" minimum:"100" maximum:"100" example:"100"`
	DurationInMonths int `json:"durationInMonths" binding:"required,gte=1" minimum:"1" example:"12"`
}

type upgradeAccountObject struct {
	MetadataKeys     []string `json:"metadataKeys" binding:"required" example:"an array containing all your metadata keys"`
	FileHandles      []string `json:"fileHandles" binding:"required" example:"an array containing all your file handles"`
	StorageLimit     int      `json:"storageLimit" binding:"required,gte=100" minimum:"100" maximum:"100" example:"100"`
	DurationInMonths int      `json:"durationInMonths" binding:"required,gte=1" minimum:"1" example:"12"`
}

type getUpgradeAccountInvoiceReq struct {
	verification
	requestBody
	getUpgradeAccountInvoiceObject getUpgradeAccountInvoiceObject
}

type getUpgradeAccountInvoiceRes struct {
	OpqInvoice models.Invoice `json:"opqInvoice"`
	UsdInvoice float64        `json:"usdInvoice"`
}

func (v *getUpgradeAccountInvoiceReq) getObjectRef() interface{} {
	return &v.getUpgradeAccountInvoiceObject
}

func GetAccountUpgradeInvoiceHandler() gin.HandlerFunc {
	return ginHandlerFunc(getAccountUpgradeInvoice)
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
