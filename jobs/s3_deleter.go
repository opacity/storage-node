package jobs

import (
	"time"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type s3Deleter struct {
}

func (e s3Deleter) Name() string {
	return "s3Deleter"
}

func (e s3Deleter) ScheduleInterval() string {
	return "@midnight"
}

func (e s3Deleter) Run() {
	utils.SlackLog("running " + e.Name())

	fileIDs, err := models.GetAllExpiredCompletedFiles(time.Now().Add(-24 * time.Hour * 60))
	if err != nil {
		utils.LogIfError(err, nil)
		return
	}

	var fileDatas, metadatas, publicShares, publicSharesThumbnail []string
	for _, fileID := range fileIDs {
		fileDatas = append(fileDatas, models.GetFileDataKey(fileID))
		metadatas = append(metadatas, models.GetFileMetadataKey(fileID))
		publicShares = append(publicShares, models.GetFileDataPublicKey(fileID))
		publicSharesThumbnail = append(publicSharesThumbnail, models.GetPublicThumbnailKey(fileID))
	}

	if err := utils.DeleteDefaultBucketObjects(fileDatas); err != nil {
		utils.LogIfError(err, nil)
		return
	}

	if err := utils.DeleteDefaultBucketObjects(metadatas); err != nil {
		utils.LogIfError(err, nil)
		return
	}

	if err := utils.DeleteDefaultBucketObjects(publicShares); err != nil {
		utils.LogIfError(err, nil)
		return
	}

	if err := utils.DeleteDefaultBucketObjects(publicSharesThumbnail); err != nil {
		utils.LogIfError(err, nil)
		return
	}

	err = models.DeleteAllCompletedFiles(fileIDs)
	utils.LogIfError(err, nil)

	err = models.RemovePublicSharesByIds(fileIDs)
	utils.LogIfError(err, nil)
}

func (e s3Deleter) Runnable() bool {
	return models.DB != nil
}
