package main

import (
	"bytes"
	"flag"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/opacity/storage-node/jobs"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/routes"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

func main() {
	defer catchError()
	defer models.Close()
	migrateFromBadger := flag.Bool("migrateFromBadger", false, "set this flag to execute the migration from BadgerDB to DynamoDB; if the migration was executed already, it will do nothing")
	flag.Parse()
	utils.SetLive()

	if *migrateFromBadger {
		err := utils.MigrateFromBadger()
		utils.PanicOnError(err)
	}

	services.SetWallet()
	err := services.InitStripe()
	utils.PanicOnError(err)

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

func catchError() {
	// Capture the error
	if r := recover(); r != nil {
		buff := bytes.NewBufferString("")
		buff.Write(debug.Stack())
		stacks := strings.Split(buff.String(), "\n")

		threadId := stacks[0]
		if len(stacks) > 5 {
			stacks = stacks[5:] // skip the Stack() and Defer method.
		}
		utils.SlackLogError(fmt.Sprintf("Crash due to err %v!!!\nRunning on thread: %s,\nStack: \n%v\n", r, threadId, strings.Join(stacks, "\n")))
	}
}
