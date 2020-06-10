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
	/** Indicate whether it can run or not. */
	Runnable() bool
	Name() string
}

type StartUpRunnable interface {
	Run() error
}

func StartupJobs() {
	utils.SlackLog("Running Startup Jobs")
	defer utils.SlackLog("Finished Startup Jobs")

	jobs := []StartUpRunnable{
		noOps{},
		s3LifeCycleSetup{},
	}

	for _, s := range jobs {
		err := s.Run()
		if err != nil {
			utils.PanicOnError(errors.New(fmt.Sprintf("Abort!!!! Unable to startup process with error: %s", err)))
		}
	}

	// Run metric collector immediately upon startup so we don't have to wait 24 hours everytime we deploy
	// TODO:  change BackgroundRunnable job's Run() methods to also return an error, so that we can have jobs
	// that we run both at startup and on a schedule
	metricCollector{}.Run()
}

func ScheduleBackgroundJobs() {
	jobrunner.Start()
	jobs := []BackgroundRunnable{
		&pingStdOut{counter: 1},
		s3Deleter{},
		s3ExpireAccess{},
		metricCollector{},
		unpaidAccountDeleter{},
		tokenCollector{},
		fileCleaner{},
		stripePaymentDeleter{},
		upgradeDeleter{},
		renewalDeleter{},
		expiredAccountDeleter{},
		badgerGarbageCollectionRunner{},
	}

	for _, s := range jobs {
		if s.Runnable() {
			utils.SlackLog("job " + s.Name() + " is runnable")
			jobrunner.Schedule(s.ScheduleInterval(), s)
		} else {
			utils.LogIfError(errors.New("job "+s.Name()+" not runnable"), nil)
		}
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

func (e *pingStdOut) Name() string {
	return "pingStdOut"
}

func (e *pingStdOut) ScheduleInterval() string {
	return "@every 60s"
}

func (e *pingStdOut) Run() {
	utils.GetLogger("jobs-ping-std-out").Infof("Pinging with count %d\n", e.counter)
	e.counter = e.counter + 1
	utils.Metrics_PingStdOut_Counter.Inc()
}

func (e *pingStdOut) Runnable() bool {
	return true
}

type noOps struct{}

func (e noOps) Run() error {
	utils.GetLogger("jobs-no-ops").Info("Run noOps")
	return nil
}
