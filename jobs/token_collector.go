package jobs

import (
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

type tokenCollector struct{}

func (t tokenCollector) ScheduleInterval() string {
	return "@every 30m"
}

func (t tokenCollector) Run() {
	for paymentStatus := models.InitialPaymentInProgress; paymentStatus < models.PaymentRetrievalComplete; paymentStatus++ {
		accounts := models.GetAccountsByPaymentStatus(paymentStatus)
		runCollectionSequence(accounts)
	}
}

func (t tokenCollector) Runnable() bool {
	err := services.SetWallet()
	return models.DB != nil && err == nil
}

func runCollectionSequence(accounts []models.Account) {
	for _, account := range accounts {
		if utils.FreeModeEnabled() {
			err := models.DB.Model(&account).Update("payment_status", models.PaymentRetrievalComplete).Error
			utils.LogIfError(err, nil)
			continue
		}
		models.PaymentCollectionFunctions[account.PaymentStatus](account)
	}
}
