package routes

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
	sia_types "gitlab.com/NebulousLabs/Sia/types"
)

type AdminSiaAllowance struct {
	AllowanceFunds   string
	ExpectedStorage  string
	Period           string
	Hosts            uint64
	RenewWindow      string
	ExpectedDownload string
	ExpectedUpload   string
}

func AdminSiaAllowanceGetHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminSiaAllowanceGet)
}

func AdminSiaAllowanceChangeHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminSiaAllowanceChange)
}

func adminSiaAllowanceGet(c *gin.Context) error {
	return siaAllowanceGet(c, "")
}

func adminSiaAllowanceChange(c *gin.Context) error {
	defer c.Request.Body.Close()

	err := c.Request.ParseForm()
	if err != nil {
		return BadRequestResponse(c, errors.New("something went wrong"))
	}

	allowanceFunds := c.Request.Form.Get("allowance-funds")
	expectedStorageTB := c.Request.Form.Get("expected-storage")
	period := c.Request.Form.Get("period")
	hosts := c.Request.Form.Get("hosts")
	renewWindowWeeks := c.Request.Form.Get("renew-window")
	expectedDownload := c.Request.Form.Get("expected-download")
	expectedUpload := c.Request.Form.Get("expected-upload")

	if err := utils.SetSiaAllowance(allowanceFunds, period, hosts, renewWindowWeeks, expectedStorageTB, expectedDownload, expectedUpload); err != nil {
		return InternalErrorResponse(c, err)
	}

	return siaAllowanceGet(c, "Allowance set successfully")
}

func siaAllowanceGet(c *gin.Context, notificationMessage string) error {
	renter := utils.GetSiaRenter()

	fundsH := renter.Settings.Allowance.Funds.Div64(1e18)
	funds := fundsH.Div64(1e6).String()

	adminSiaAllowance := AdminSiaAllowance{
		AllowanceFunds:   funds,
		ExpectedStorage:  fmt.Sprintf("%.2f", float64(renter.Settings.Allowance.ExpectedStorage)/1e12),
		Period:           fmt.Sprintf("%.1f", float64(renter.Settings.Allowance.Period)/float64(sia_types.BlocksPerWeek)),
		Hosts:            renter.Settings.Allowance.Hosts,
		RenewWindow:      fmt.Sprintf("%.1f", float64(renter.Settings.Allowance.RenewWindow)/float64(sia_types.BlocksPerWeek)),
		ExpectedDownload: fmt.Sprintf("%.2f", float64(renter.Settings.Allowance.ExpectedDownload)*float64(sia_types.BlocksPerMonth)/1e12),
		ExpectedUpload:   fmt.Sprintf("%.2f", float64(renter.Settings.Allowance.ExpectedUpload)*float64(sia_types.BlocksPerMonth)/1e12),
	}

	totalSpent, unspentAllocated, unspentUnallocated := renter.FinancialMetrics.SpendingBreakdown()

	walletInfo := utils.GetWalletInfo()

	c.HTML(http.StatusOK, "sia-allowance.tmpl", gin.H{
		"title":                         "Sia allowance",
		"allowance":                     adminSiaAllowance,
		"totalSpent":                    totalSpent.HumanString(),
		"contractFees":                  renter.FinancialMetrics.ContractFees.HumanString(),
		"storageSpending":               renter.FinancialMetrics.StorageSpending.HumanString(),
		"unspentAllocated":              unspentAllocated.HumanString(),
		"unspentUnallocated":            unspentUnallocated.HumanString(),
		"currentPeriod":                 renter.CurrentPeriod,
		"nextPeriod":                    renter.NextPeriod,
		"walletConfirmedSiacoinBalance": walletInfo.ConfirmedSiacoinBalance.HumanString(),
		"notificationMessage":           "",
	})

	return nil
}
