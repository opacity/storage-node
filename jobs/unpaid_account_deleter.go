package jobs

import (
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type unpaidAccountDeleter struct {
}

func (u unpaidAccountDeleter) Name() string {
	return "unpaidAccountDeleter"
}

func (u unpaidAccountDeleter) ScheduleInterval() string {
	return "@every 8h"
}

func (u unpaidAccountDeleter) Run() {
	utils.SlackLog("running " + u.Name())

	err := models.PurgeOldUnpaidAccounts(utils.Env.AccountRetentionDays)

	utils.LogIfError(err, nil)
}

func (u unpaidAccountDeleter) Runnable() bool {
	return models.DB != nil
}
