package jobs

import (
	"time"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type expiredAccountDeleter struct {
}

func (e expiredAccountDeleter) Name() string {
	return "expiredAccountDeleter"
}

func (e expiredAccountDeleter) ScheduleInterval() string {
	return "@midnight"
}

func (e expiredAccountDeleter) Run() {
	utils.SlackLog("running " + e.Name())

	models.DeleteExpiredAccounts(time.Now().Add(-24 * time.Hour * 60))
}

func (e expiredAccountDeleter) Runnable() bool {
	return models.DB != nil
}
