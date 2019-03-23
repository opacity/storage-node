package jobs

import (
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type unpaidAccountDeleter struct {
}

func (u unpaidAccountDeleter) ScheduleInterval() string {
	return "@midnight"
}

func (u unpaidAccountDeleter) Run() {
	err := models.PurgeOldUnpaidAccounts(utils.Env.AccountRetentionDays)

	utils.LogIfError(err, nil)
}

func (u unpaidAccountDeleter) Runnable() bool {
	return models.DB != nil
}
