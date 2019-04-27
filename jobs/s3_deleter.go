package jobs

import (
	"time"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type s3Deleter struct {
}

func (e s3Deleter) ScheduleInterval() string {
	return "@midnight"
}

func (e s3Deleter) Run() {
	fileIDs, err := models.GetAllExpiredCompletedFiles(time.Now())
	if err != nil {
		utils.LogIfError(err, nil)
		return
	}

	if err := utils.DeleteDefaultBucketObjects(fileIDs); err != nil {
		utils.LogIfError(err, nil)
		return
	}

	err = models.DeleteAllCompletedFiles(fileIDs)
	utils.LogIfError(err, nil)
}

func (e s3Deleter) Runnable() bool {
	return models.DB != nil
}
