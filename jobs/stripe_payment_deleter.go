package jobs

import (
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type stripePaymentDeleter struct{}

func (s stripePaymentDeleter) Name() string {
	return "stripePaymentDeleter"
}

func (s stripePaymentDeleter) ScheduleInterval() string {
	return "@midnight"
}

func (s stripePaymentDeleter) Run() {
	utils.SlackLog("running " + s.Name())

	err := models.PurgeOldStripePayments(utils.Env.StripeRetentionDays)

	utils.LogIfError(err, nil)
}

func (s stripePaymentDeleter) Runnable() bool {
	// @TODO: investigate this, why is wallet needed here?
	// err := services.SetWallets()
	return models.DB != nil
}
