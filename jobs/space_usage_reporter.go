package jobs

import (
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type spaceUsageReporter struct{}

func (s spaceUsageReporter) ScheduleInterval() string {
	return "@every 24h"
}

func (s spaceUsageReporter) Run() {
	spaceReport := models.CreateSpaceUsedReport()
	spaceUsed := (spaceReport.SpaceUsedSum / float64(spaceReport.SpaceAllotedSum)) * float64(100)

	utils.Metrics_Percent_Of_Space_Used.Set(spaceUsed)
}
