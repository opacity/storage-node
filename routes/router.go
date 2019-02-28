package routes

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/jobs"
	"github.com/opacity/storage-node/utils"
)

var uptime time.Time

const (
	/*AccountsPath is the path for dealing with accounts*/
	AccountsPath = "/accounts"

	/*V1Path is a router group for the v1 version of storage node*/
	V1Path = "/api/v1"

	/*AdminPath is a router group for admin task. */
	AdminPath = "/admin"
)

func init() {
}

/*CreateRoutes creates our application's routes*/
func CreateRoutes() {
	uptime = time.Now()

	router := returnEngine()

	setupV1Paths(returnV1Group(router))
	setupAdminPaths(router)

	// Listen and Serve
	err := router.Run(":" + os.Getenv("PORT"))
	utils.LogIfError(err)
}

func returnEngine() *gin.Engine {
	router := gin.Default()
	config := cors.DefaultConfig()

	// TODO:  update to only allow our frontend and localhost
	config.AllowAllOrigins = true
	router.Use(cors.New(config))

	// Test app is running
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]interface{}{
			"message": "Storage node is running",
			"uptime":  fmt.Sprintf("%v", time.Now().Sub(uptime)),
		})
	})
	return router
}

func returnV1Group(router *gin.Engine) *gin.RouterGroup {
	return router.Group(V1Path)
}

func setupV1Paths(v1Router *gin.RouterGroup) {
	v1Router.POST(AccountsPath, CreateAccountHandler())
	v1Router.GET(AccountsPath+"/:accountID", CheckAccountPaymentStatusHandler())

	v1Router.POST("/trial-upload", func(c *gin.Context) {
		c.JSON(http.StatusOK, "stub for doing a trial upload")
	})

	v1Router.POST("/upload", UploadFileHandler())
	v1Router.GET("/download/:accountID/:uploadID", DownloadFileHandler())
}

func setupAdminPaths(router *gin.Engine) {
	g := router.Group(AdminPath)
	g.GET("/jobrunner/json", jobs.JobJson)

	// Load template file location relative to the current working directory
	// Unable to find the file.
	// g.GET("/jobrunner/html", jobs.JobHtml)
	//router.LoadHTMLGlob("../../bamzi/jobrunner/views/Status.html")
}
