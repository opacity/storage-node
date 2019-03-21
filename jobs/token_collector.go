package jobs

import (
	"github.com/opacity/storage-node/models"
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

func runCollectionSequence(accounts []models.Account) {
	for _, account := range accounts {
		models.PaymentCollectionFunctions[account.PaymentStatus](account)
	}
}
