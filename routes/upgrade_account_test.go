package routes

import (
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func Test_Init_Upgrade_Accounts(t *testing.T) {
	setupTests(t)
}

func Test_GetAccountUpgradeInvoiceHandler_Returns_Invoice(t *testing.T) {
	getInvoiceObj := getUpgradeAccountInvoiceObject{
		StorageLimit:     2048,
		DurationInMonths: 12,
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getInvoiceObj)

	getInvoiceReq := getUpgradeAccountInvoiceReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)

	account.StorageLimit = models.StorageLimitType(1024)
	account.CreatedAt = time.Now().Add(time.Hour * 24 * (365 / 2) * -1)
	models.DB.Save(&account)

	w := httpPostRequestHelperForTest(t, AccountUpgradeInvoicePath, getInvoiceReq)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"usdInvoice":100`)
	assert.Contains(t, w.Body.String(), `"opqInvoice":{"cost":24,`)
}
