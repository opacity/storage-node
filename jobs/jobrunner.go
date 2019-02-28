package jobs

import (
	"fmt"

	"github.com/bamzi/jobrunner"
	"github.com/gin-gonic/gin"
)

func StartJobs() {
	jobrunner.Start()
	jobs := []JobRunnable{
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

type JobRunnable interface {
	ScheduleInterval() string
	Run()
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
