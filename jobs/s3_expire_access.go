package jobs

import (
	"time"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type s3ExpireAccess struct {
}

func (e s3ExpireAccess) ScheduleInterval() string {
	return "@every 1h"
}

func (e s3ExpireAccess) Run() {
	expired := make([]models.S3ObjectLifeCycle, 0)
	if err := models.DB.Where("expired_time > ?", time.Now()).Find(&expired).Error; err != nil {
		fmt.Println("Some error occurred")
	}

	for _, v := range expired {

	}
	models.DB.Delete(&expired)
}
