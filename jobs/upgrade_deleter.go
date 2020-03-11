package jobs

import (
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type upgradeDeleter struct{}

const hoursToRetainUpgrade = 4

func (u upgradeDeleter) Name() string {
	return "upgradeDeleter"
}

func (u upgradeDeleter) ScheduleInterval() string {
	return "@every 30m"
}

func (u upgradeDeleter) Run() {
	utils.SlackLog("running " + u.Name())

	err := models.PurgeOldUpgrades(hoursToRetainUpgrade)

	utils.LogIfError(err, nil)
}

func (u upgradeDeleter) Runnable() bool {
	return models.DB != nil
}
