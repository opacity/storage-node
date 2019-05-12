package jobs

import (
	"time"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type fileCleaner struct {
}

var newerThanOffset = -5 * time.Minute
var olderThanOffset = -1 * 24 * time.Hour

func (f fileCleaner) ScheduleInterval() string {
	return "@every 15s"
}

func (f fileCleaner) Run() {
	err := models.DeleteUploadsOlderThan(time.Now().Add(olderThanOffset))
	utils.LogIfError(err, nil)
}

func (f fileCleaner) Runnable() bool {
	return models.DB != nil
}
