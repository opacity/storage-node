package jobs

import (
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

type stripePaymentDeleter struct{}

func (s stripePaymentDeleter) ScheduleInterval() string {
	return "@midnight"
}

func (s stripePaymentDeleter) Run() {
	err := models.PurgeOldStripePayments(utils.Env.StripeRetentionDays)

	utils.LogIfError(err, nil)
}

func (s stripePaymentDeleter) Runnable() bool {
	err := services.SetWallet()
	return models.DB != nil && err == nil
}
