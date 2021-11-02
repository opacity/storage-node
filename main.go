package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/opacity/storage-node/jobs"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/routes"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

var GO_ENV = "localhost"
var VERSION = "local"

func main() {
	if GO_ENV == "" {
		utils.PanicOnError(errors.New("the GO_ENV variable is not set; application can not run"))
	}
	os.Setenv("GO_ENV", GO_ENV)
	tracesSampleRate := 0.0
	if GO_ENV == "production" {
		tracesSampleRate = 0.25
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              "https://03e807e8312d47938a94b73ebec3cc84@o126495.ingest.sentry.io/5855671",
			Release:          VERSION,
			Environment:      GO_ENV,
			AttachStacktrace: true,
			TracesSampleRate: tracesSampleRate,
			BeforeSend:       sentryOpacityBeforeSend,
		})
		if err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}
		defer sentry.Flush(5 * time.Second)
	}

	defer catchError()
	defer models.Close()

	utils.SetLive()

	err := services.InitStripe(utils.Env.StripeKeyProd)
	utils.PanicOnError(err)

	utils.SlackLog("Begin to restart service!")

	if !utils.Env.DisableDbConn {
		models.Connect(utils.Env.DatabaseURL)
	}

	migratePlanIds := utils.GetPlansMigrationDone()
	if !migratePlanIds {
		err = models.MigratePlanIds()
		utils.PanicOnError(err)
	}

	jobs.CreatePlanMetrics()

	models.MigrateEnvWallets()

	jobs.StartupJobs()
	if utils.Env.EnableJobs {
		jobs.ScheduleBackgroundJobs()
	}

	routes.CreateRoutes()
}

func sentryOpacityBeforeSend(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	if event.Request != nil {
		req := routes.GenericRequest{}

		if err := json.Unmarshal([]byte(event.Request.Data), &req); err == nil {
			if len(event.Exception) > 0 {
				frames := event.Exception[0].Stacktrace.Frames
				// do not include http/gin-gonic and the Sentry throw funcs ones
				event.Exception[0].Stacktrace.Frames = frames[6 : len(frames)-3]
			}

			event.Request.Data = req.RequestBody
		}
	}

	return event
}

func catchError() {
	// Capture the error
	if r := recover(); r != nil {
		sentry.CurrentHub().Recover(r)

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
