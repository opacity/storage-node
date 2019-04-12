package jobs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type userStatsSetup struct{}

func (e userStatsSetup) Run() error {
	count := 0
	models.DB.Model(&models.Account).Count(&count)

	diff := count - int(utils.GetMetricCounter(utils.Metrics_AccountCreated_Counter))
	utils.Metrics_AccountCreated_Counter.Add(float64(diff))

	fileStats := s3FileStats{}
	return fileStats.updateCount()
}
