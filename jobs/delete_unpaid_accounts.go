package jobs

import (
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type accountDeleter struct {
}

func (a accountDeleter) ScheduleInterval() string {
	return "@midnight"
}

func (a accountDeleter) Run() {
	err := models.PurgeOldUnpaidAccounts(utils.Env.AccountRetentionDays)

	utils.LogIfError(err, nil)
}
