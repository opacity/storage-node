package jobs

import (
	"errors"
	"fmt"

	"github.com/bamzi/jobrunner"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

type BackgroundRunnable interface {
	ScheduleInterval() string
	Run()
}

type StartUpRunnable interface {
	Run() error
}

func StartupJobs() {
	utils.SlackLog("Run StartUp Jobs")
	defer utils.SlackLog("Finished StartUp Jobs")

	jobs := []StartUpRunnable{
		noOps{},
		s3LifeCycle{},
	}

	for _, s := range jobs {
		err := s.Run()
		if err != nil {
			utils.PanicOnError(errors.New(fmt.Sprintf("Abort!!!! Unable to startup process with error: %s", err)))
		}
	}
}

func ScheduleBackgroundJobs() {
	jobrunner.Start()
	jobs := []BackgroundRunnable{
		&pingStdOut{counter: 1},
		s3Deleter{},
	}

	for _, s := range jobs {
		jobrunner.Schedule(s.ScheduleInterval(), s)
	}
}

func JobJson(c *gin.Context) {
	// returns a map[string]interface{} that can be marshalled as JSON
	c.JSON(200, jobrunner.StatusJson())
}

func JobHtml(c *gin.Context) {
	// Returns the template data pre-parsed
	c.HTML(200, "", jobrunner.StatusPage())
}

type pingStdOut struct {
	counter int
}

func (e *pingStdOut) ScheduleInterval() string {
	return "@every 60s"
}

func (e *pingStdOut) Run() {
	fmt.Printf("Pinging with count %d\n", e.counter)
	e.counter = e.counter + 1
}

type noOps struct {
}

func (e noOps) Run() error {
	return nil
}
