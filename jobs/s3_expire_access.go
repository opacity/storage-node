package jobs

import (
	"time"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type s3ExpireAccess struct{}

func (e s3ExpireAccess) ScheduleInterval() string {
	return "@every 1h"
}

func (e s3ExpireAccess) Run() {
	expired := make([]models.S3ObjectLifeCycle, 0)
	if err := models.DB.Where("expired_time < ?", time.Now()).Find(&expired).Error; err != nil {
		utils.GetLogger("jobs-s3-expire-access").Errorf("Some error occur on querying DB: %v", err)
		return
	}

	for _, v := range expired {
		if err := utils.DeleteDefaultBucketObject(v.ObjectName); err == nil {
			models.DB.Delete(&v)
		}
	}
}
