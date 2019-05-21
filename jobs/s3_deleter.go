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

	var fileDatas []string
	for _, fileID := range fileIDs {
		fileDatas = append(fileDatas, models.GetFileDataKey(fileID))
	}

	if err := utils.DeleteDefaultBucketObjects(fileDatas); err != nil {
		utils.LogIfError(err, nil)
		return
	}

	var metadatas []string
	for _, fileID := range fileIDs {
		metadatas = append(metadatas, models.GetFileMetadataKey(fileID))
	}

	if err := utils.DeleteDefaultBucketObjects(metadatas); err != nil {
		utils.LogIfError(err, nil)
		return
	}

	err = models.DeleteAllCompletedFiles(fileIDs)
	utils.LogIfError(err, nil)
}

func (e s3Deleter) Runnable() bool {
	return models.DB != nil
}
