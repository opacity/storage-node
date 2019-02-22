package routes

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

var uptime time.Time

/*AccountsPath is the path for dealing with accounts*/
const AccountsPath = "/accounts"

/*V1Path is a router group for the v1 version of storage node*/
const V1Path = "/api/v1"

func init() {
}

/*CreateRoutes creates our application's routes*/
func CreateRoutes() {
	uptime = time.Now()

	router := returnEngine()

	v1 := returnV1Group(router)

	setupV1Paths(v1)

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

	v1Router.POST("/uploads", UploadFileHandler())
	v1Router.POST("/new-upload", func(c *gin.Context) {
		c.JSON(http.StatusOK, "stub for uploading a file with an existing subscription")
	})
}
