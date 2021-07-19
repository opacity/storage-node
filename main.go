package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/opacity/storage-node/jobs"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/routes"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

var GO_ENV string

func main() {
	err := sentry.Init(sentry.ClientOptions{
		// Dsn:         "https://03e807e8312d47938a94b73ebec3cc84@o126495.ingest.sentry.io/5855671",
		// @TODO: replace Dsn, below is a sandbox environment (Mihai)
		Dsn:         "https://2603daa7015a4decb2ca5c76876bb1e8@o920906.ingest.sentry.io/5866955",
		Release:     "storage-node@2.0",
		Environment: GO_ENV,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
	sentry.CaptureMessage("is sentry set correctly?")
	sentry.CaptureException(errors.New("arbitrary error to see stack trace"))

	defer catchError()
	defer models.Close()

	utils.SetLive()
	services.SetWallet()
	err = services.InitStripe()
	utils.PanicOnError(err)

	utils.SlackLog("Begin to restart service!")

	if !utils.Env.DisableDbConn {
		models.Connect(utils.Env.DatabaseURL)
	}

	setEnvPlans()

	jobs.StartupJobs()
	if utils.Env.EnableJobs {
		jobs.ScheduleBackgroundJobs()
	}

	routes.CreateRoutes()
}

func setEnvPlans() {
	plans := []utils.PlanInfo{}
	results := models.DB.Find(&plans)

	utils.Env.Plans = make(utils.PlanResponseType)

	if results.RowsAffected == 0 {
		err := json.Unmarshal([]byte(utils.DefaultPlansJson), &utils.Env.Plans)
		utils.LogIfError(err, nil)

		for _, plan := range utils.Env.Plans {
			models.DB.Model(&utils.PlanInfo{}).Create(&plan)
		}
	} else {
		for _, plan := range plans {
			utils.Env.Plans[plan.StorageInGB] = plan
		}
	}

	utils.CreatePlanMetrics()
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
