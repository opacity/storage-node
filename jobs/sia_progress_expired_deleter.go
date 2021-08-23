package jobs

import (
	"time"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type siaProgressExpiredDeleter struct{}

func (j siaProgressExpiredDeleter) Name() string {
	return "siaProgressExpiredDeleter"
}

func (j siaProgressExpiredDeleter) ScheduleInterval() string {
	return "@every 36h"
}

func (j siaProgressExpiredDeleter) Run() {
	err := models.DeleteAllExpiredSiaProgressFiles(time.Now().Add(-36 * time.Hour))
	utils.LogIfError(err, nil)
}

func (j siaProgressExpiredDeleter) Runnable() bool {
	return models.DB != nil
}
