package jobs

import (
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

type tokenCollector struct{}

func (t tokenCollector) Name() string {
	return "tokenCollector"
}

func (t tokenCollector) ScheduleInterval() string {
	return "@every 30m"
}

func (t tokenCollector) Run() {
	utils.SlackLog("running " + t.Name())
	for paymentStatus := models.InitialPaymentInProgress; paymentStatus < models.PaymentRetrievalComplete; paymentStatus++ {
		accounts := models.GetAccountsByPaymentStatus(paymentStatus)
		runCollectionSequence(accounts)
	}
}

func (t tokenCollector) Runnable() bool {
	err := services.SetWallet()
	utils.LogIfError(err, nil)
	return models.DB != nil && err == nil
}

func runCollectionSequence(accounts []models.Account) {
	for _, account := range accounts {
		err := models.PaymentCollectionFunctions[account.PaymentStatus](account)
		cost, _ := account.Cost()
		utils.LogIfError(err, map[string]interface{}{
			"eth_address":    account.EthAddress,
			"account_id":     account.AccountID,
			"payment_status": models.PaymentStatusMap[account.PaymentStatus],
			"cost":           cost,
		})
	}
}
