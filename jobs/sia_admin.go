package jobs

import (
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

type siaAdmin struct{}

func AdminSiaStatsHandler(c *gin.Context) {
	rg := utils.GetSiaRenter()
	walletInfo := utils.GetWalletInfo()
	totalSpent, unspentAllocated, unspentUnallocated := rg.FinancialMetrics.SpendingBreakdown()

	unspentUnallocatedFloat, _ := unspentUnallocated.Float64()
	allowanceFundsFloat, _ := rg.Settings.Allowance.Funds.Float64()

	unspentUnallocatedOutOfAllowancePerc := (unspentUnallocatedFloat / allowanceFundsFloat) * 100

	c.JSON(200, map[string]interface{}{
		"unspentUnallocatedOutOfAllowancePerc": unspentUnallocatedOutOfAllowancePerc,
		"Wallet Confirmed Balance":             walletInfo.ConfirmedSiacoinBalance,
		"Wallet Confirmed Balance SC":          walletInfo.ConfirmedSiacoinBalance.HumanString(),
		"Allowance":                            rg.Settings.Allowance.Funds,
		"Allowance SC":                         rg.Settings.Allowance.Funds.HumanString(),
		"Total Spent":                          totalSpent,
		"Total Spent SC":                       totalSpent.HumanString(),
		"Unspent Allocated":                    unspentAllocated,
		"Unspent Allocated SC":                 unspentAllocated.HumanString(),
		"Unspent Unallocated":                  unspentUnallocated,
		"Unspent Unallocated SC":               unspentUnallocated.HumanString(),
		"Renter":                               rg,
	})
}

func (s siaAdmin) Name() string {
	return "siaAdmin"
}

func (s siaAdmin) ScheduleInterval() string {
	return "@midnight"
}

func (s siaAdmin) Run() {
	utils.SlackLog("running " + s.Name())

}

func (s siaAdmin) Runnable() bool {
	if err := utils.IsSiaClientInit(); err != nil {
		return false
	}
	return true
}
