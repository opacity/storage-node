package jobs

import (
	"time"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type expiredCompletedFilesDeleter struct {
}

func (e expiredCompletedFilesDeleter) Name() string {
	return "expiredCompletedFilesDeleter"
}

func (e expiredCompletedFilesDeleter) ScheduleInterval() string {
	return "@midnight"
}

func (e expiredCompletedFilesDeleter) Run() {
	utils.SlackLog("running " + e.Name())

	expiredCompletedFiles, err := models.GetAllExpiredCompletedFiles(time.Now().Add(-24 * time.Hour * 60))
	if err != nil {
		utils.LogIfError(err, nil)
		return
	}

	var fileIDs, s3FileDatas, s3Metadatas, publicShares, publicSharesThumbnail []string
	for _, expiredCompletedFile := range expiredCompletedFiles {
		fileIDs = append(fileIDs, expiredCompletedFile.FileID)
		if expiredCompletedFile.StorageType == utils.S3 {
			s3FileDatas = append(s3FileDatas, models.GetFileDataKey(expiredCompletedFile.FileID))
			s3Metadatas = append(s3Metadatas, models.GetFileMetadataKey(expiredCompletedFile.FileID))
			publicShares = append(publicShares, models.GetFileDataPublicKey(expiredCompletedFile.FileID))
			publicSharesThumbnail = append(publicSharesThumbnail, models.GetPublicThumbnailKey(expiredCompletedFile.FileID))
		}

		if expiredCompletedFile.StorageType == utils.Sia {
			utils.DeleteSiaFile(expiredCompletedFile.FileID)
		}
	}

	if len(s3FileDatas) > 0 {
		if err := utils.DeleteDefaultBucketObjects(s3FileDatas); err != nil {
			utils.LogIfError(err, nil)
			return
		}
	}

	if len(s3Metadatas) > 0 {
		if err := utils.DeleteDefaultBucketObjects(s3Metadatas); err != nil {
			utils.LogIfError(err, nil)
			return
		}
	}

	if len(publicShares) > 0 {
		if err := utils.DeleteDefaultBucketObjects(publicShares); err != nil {
			utils.LogIfError(err, nil)
			return
		}
	}

	if len(publicSharesThumbnail) > 0 {
		if err := utils.DeleteDefaultBucketObjects(publicSharesThumbnail); err != nil {
			utils.LogIfError(err, nil)
			return
		}
	}

	err = models.DeleteAllCompletedFiles(fileIDs)
	utils.LogIfError(err, nil)

	err = models.RemovePublicSharesByIds(fileIDs)
	utils.LogIfError(err, nil)
}

func (e expiredCompletedFilesDeleter) Runnable() bool {
	return models.DB != nil
}
