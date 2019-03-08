package main

import (
	"github.com/opacity/storage-node/jobs"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/routes"
	"github.com/opacity/storage-node/utils"
)

func main() {
	defer models.Close()

	//utils.SetProduction()
	utils.SetDevelopment()

	utils.SlackLog("Begin to restart service!")

	if !utils.Env.DisableDbConn {
		models.Connect(utils.Env.DatabaseURL)
	}

	jobs.StartupJobs()
	if utils.Env.EnableJobs {
		jobs.ScheduleBackgroundJobs()
	}

	routes.CreateRoutes()
}
