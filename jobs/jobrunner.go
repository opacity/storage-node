package jobs

import (
	"fmt"

	"github.com/bamzi/jobrunner"
	"github.com/gin-gonic/gin"
)

func StartJobs() {
	jobrunner.Start()
	jobrunner.Schedule("@every 5s", &PingStdOut{counter: 1})
}

func JobJson(c *gin.Context) {
	// returns a map[string]interface{} that can be marshalled as JSON
	c.JSON(200, jobrunner.StatusJson())
}

func JobHtml(c *gin.Context) {
	// Returns the template data pre-parsed
	c.HTML(200, "", jobrunner.StatusPage())

}

type PingStdOut struct {
	counter int
}

func (e *PingStdOut) Run() {
	fmt.Printf("Pinging with count %d", e.counter)
	e.counter = e.counter + 1
}
