package jobs

import (
	"time"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type s3FileStats struct{}

func (e s3FileStats) ScheduleInterval() string {
	return "@every 24h"
}

func (e s3FileStats) Run() {
	e.updateCount()
}

func (e s3ExpireAccess) Runnable() bool {
	return utils.IsS3Enabled()
}

func (e s3FileStats) updateCount() error {
	// this is suck since S3 does not provide any API to do, and this is an expensive operation
	// Since file can be expired on S3 side while we did not trace it. The only way is to iterator
	// all data and figure it.

	fileCounter := 0
	byteSizeCounter := int64(0)
	err := utils.IterateDefaultBucketAllObjects(func(objs []*s3.Object) bool {
		fileCounter += len(objs)
		for _, c := range objs {
			byteSizeCounter += aws.Int64Value(c.Size)
		}
		return true
	})

	fileDiff := fileCounter - int(utils.GetMetricCounter(utils.Metrics_FileUploaded_Counter))
	utils.Metrics_FileUploaded_Counter.Add(float64(fileDiff))

	byteDiff := byteSizeCounter - int64(utils.GetMetricCounter(Metrics_FileUploadedSizeInByte_Counter))
	utils.Metrics_FileUploadedSizeInByte_Counter.Add(float64(byteDiff))

	return err
}
