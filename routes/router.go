package routes

import (
	"fmt"
	"net/http"
	"os"
	"time"

	_ "github.com/opacity/storage-node/docs"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/jobs"
	"github.com/opacity/storage-node/utils"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
)

// @title Storage Node
// @version 1.0
// @description Opacity backend for file storage.
// @termsOfService https://opacity.io/terms-of-service

// @contact.name Opacity Staff
// @contact.url https://telegram.me/opacitystorage

// @license.name GNU GENERAL PUBLIC LICENSE

var uptime time.Time

const (
	/*V1Path is a router group for the v1 version of storage node*/
	V1Path = "/api/v1"

	/*AccountsPath is the path for dealing with accounts*/
	AccountsPath = "/accounts"

	/*AdminPath is a router group for admin task. */
	AdminPath = "/admin"

	/*MetadataPath is the path for dealing with metadata*/
	MetadataPath = "/metadata"

	/*InitUploadPath is the path for uploading files to paid accounts*/
	InitUploadPath = "/init-upload"

	/*UploadPath is the path for uploading files to paid accounts*/
	UploadPath = "/upload"

	/*UploadStatusPath is the path for checking upload status*/
	UploadStatusPath = "/upload-status"

	/*FilePath is the path for downloading and deleting files*/
	FilePath = "/file"
)

func init() {
}

/*CreateRoutes creates our application's routes*/
func CreateRoutes() {
	uptime = time.Now()

	router := returnEngine()

	setupV1Paths(returnV1Group(router))
	setupAdminPaths(router)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Listen and Serve
	err := router.Run(":" + os.Getenv("PORT"))
	utils.LogIfError(err, map[string]interface{}{"error": err})
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
	v1Router.GET(AccountsPath, CheckAccountPaymentStatusHandler())

	v1Router.POST(MetadataPath, UpdateMetadataHandler())
	v1Router.GET(MetadataPath, GetMetadataHandler())

	v1Router.POST(InitUploadPath, InitFileUploadHandler())
	v1Router.POST(UploadPath, UploadFileHandler())
	v1Router.POST(UploadStatusPath, CheckUploadStatusHandler())

	v1Router.POST("/free_upload", FreeUploadFileHandler())
	v1Router.GET("/download", DownloadFileHandler())

	// File endpoint
	v1Router.DELETE(FilePath, DeleteFileHandler())
	v1Router.GET(FilePath, DownloadFileHandler())
}

func setupAdminPaths(router *gin.Engine) {
	g := router.Group(AdminPath)
	g.GET("/jobrunner/json", jobs.JobJson)

	g.GET("/metrics", gin.WrapH(promhttp.Handler()))

	g.GET("/user_stats", UserStatsHandler())

	// Load template file location relative to the current working directory
	// Unable to find the file.
	// g.GET("/jobrunner/html", jobs.JobHtml)
	//router.LoadHTMLGlob("../../bamzi/jobrunner/views/Status.html")
}
