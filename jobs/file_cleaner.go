package jobs

import (
	"time"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type fileCleaner struct {
}

var olderThanOffset = -1 * 24 * time.Hour

func (f fileCleaner) Name() string {
	return "fileCleaner"
}

func (f fileCleaner) ScheduleInterval() string {
	return "@every 15m"
}

func (f fileCleaner) Run() {
	utils.SlackLog("running " + f.Name())

	files, err := models.DeleteUploadsOlderThan(time.Now().Add(olderThanOffset))
	utils.LogIfError(err, nil)

	if len(files) == 0 {
		return
	}

	var metadataIDs, fileIDs []string
	for _, file := range files {
		metadataIDs = append(metadataIDs, models.GetFileMetadataKey(file.FileID))
		fileIDs = append(fileIDs, file.FileID)
	}

	utils.LogIfError(models.DeleteAllCompletedUploadIndexes(fileIDs), nil)
	utils.LogIfError(utils.DeleteDefaultBucketObjects(metadataIDs), nil)
}

func (f fileCleaner) Runnable() bool {
	return models.DB != nil
}
