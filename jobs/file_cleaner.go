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
	err := models.CompleteUploadsNewerThan(time.Now().Add(newerThanOffset))
	utils.LogIfError(err, nil)
	files, err := models.DeleteUploadsOlderThan(time.Now().Add(olderThanOffset))
	utils.LogIfError(err, nil)

	if len(files) == 0 {
		return
	}
	var ids []string
	for _, file := range files {
		ids = append(ids, models.GetFileMetadataKey(file.FileID))
	}
	err = utils.DeleteDefaultBucketObjects(ids)
	utils.LogIfError(err, nil)
}

func (f fileCleaner) Runnable() bool {
	return models.DB != nil
}
