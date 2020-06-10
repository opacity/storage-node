package jobs

import (
	"github.com/opacity/storage-node/utils"
)

type badgerGarbageCollectionRunner struct {
}

func (b badgerGarbageCollectionRunner) Name() string {
	return "badgerGarbageCollectionRunner"
}

func (b badgerGarbageCollectionRunner) ScheduleInterval() string {
	return "@midnight"
}

func (b badgerGarbageCollectionRunner) Run() {
	utils.SlackLog("running " + b.Name())

	err := utils.RunGarbageCollection()
	utils.LogIfError(err, map[string]interface{}{"process": "badgerGarbageCollectionRunner"})
}

func (b badgerGarbageCollectionRunner) Runnable() bool {
	return utils.GetBadgerDb() != nil
}
