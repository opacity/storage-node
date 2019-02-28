package jobs

import (
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type s3Deleter struct {
}

func (e s3Deleter) ScheduleInterval() string {
	return "@midnight"
}

func (e s3Deleter) Run() {
	// TODO(philip.z): figure out how to query a list of expired users.s
	// Query a list of expired account
	accounts := []models.Account{}

	for _, account := range accounts {
		utils.DeleteDefaultBucketObjectKeys(account.S3Prefix())
	}
}
