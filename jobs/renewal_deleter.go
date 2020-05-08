package jobs

import (
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type renewalDeleter struct{}

const hoursToRetainRenewal = 4

func (u renewalDeleter) Name() string {
	return "renewalDeleter"
}

func (u renewalDeleter) ScheduleInterval() string {
	return "@every 30m"
}

func (u renewalDeleter) Run() {
	utils.SlackLog("running " + u.Name())

	err := models.PurgeOldRenewals(hoursToRetainRenewal)

	utils.LogIfError(err, nil)
}

func (u renewalDeleter) Runnable() bool {
	return models.DB != nil
}
