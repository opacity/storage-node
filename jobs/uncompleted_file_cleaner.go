package jobs

import (
	"time"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type uncompletedFileCleaner struct {
}

var olderThanOffset = -1 * 24 * time.Hour

func (f uncompletedFileCleaner) Name() string {
	return "fileCleaner"
}

func (f uncompletedFileCleaner) ScheduleInterval() string {
	return "@every 6h"
}

func (f uncompletedFileCleaner) Run() {
	utils.SlackLog("running " + f.Name())

	// Sia progress files
	siaProgressFiles, err := models.DeleteAllExpiredSiaProgressFiles(time.Now().Add(olderThanOffset))
	utils.LogIfError(err, nil)
	if len(siaProgressFiles) > 0 {
		for _, siaProgressFile := range siaProgressFiles {
			utils.DeleteSiaFile(siaProgressFile.FileID)
		}
	}

	// S3 files
	s3Files, err := models.DeleteUploadsOlderThan(time.Now().Add(olderThanOffset))
	utils.LogIfError(err, nil)

	if len(s3Files) == 0 {
		return
	}

	var metadataIDs, fileIDs []string
	for _, file := range s3Files {
		metadataIDs = append(metadataIDs, models.GetFileMetadataKey(file.FileID))
		fileIDs = append(fileIDs, file.FileID)
	}

	utils.LogIfError(models.DeleteAllCompletedUploadIndexes(fileIDs), nil)
	utils.LogIfError(utils.DeleteDefaultBucketObjects(metadataIDs, utils.S3), nil)
}

func (f uncompletedFileCleaner) Runnable() bool {
	return models.DB != nil
}
