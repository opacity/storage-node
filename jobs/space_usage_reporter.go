package jobs

type spaceUsageReporter struct{}

func (s spaceUsageReporter) ScheduleInterval() string {
	return "@every 24h"
}

func (s spaceUsageReporter) Run() {
}