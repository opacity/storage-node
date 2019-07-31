package routes

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "github.com/opacity/storage-node/docs"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/jobs"
	"github.com/opacity/storage-node/services"
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

// @license.name OPACITY LIMITED CODE REVIEW LICENSE LICENSE.md

var (
	uptime time.Time

	/*EthWrapper is a copy of services.EthWrapper*/
	EthWrapper = services.EthWrapper
)

const (
	/*V1Path is a router group for the v1 version of storage node*/
	V1Path = "/api/v1"

	/*AccountsPath is the path for dealing with accounts*/
	AccountsPath = "/accounts"

	/*AccountDataPath is the path for retrieving data about an account*/
	AccountDataPath = "/account-data"

	/*AccountUpgradeInvoicePath is the path for getting an invoice to upgrade an account*/
	AccountUpgradeInvoicePath = "/upgrade/invoice"

	/*AccountUpgradePath is the path for checking the upgrade status of an account*/
	AccountUpgradePath = "/upgrade"

	/*AdminPath is a router group for admin task. */
	AdminPath = "/admin"

	/*MetadataGetPath is the path for getting metadata*/
	MetadataGetPath = "/metadata/get"

	/*MetadataHistoryPath is the path for getting historical metadata*/
	MetadataHistoryPath = "/metadata/history"

	/*MetadataSetPath is the path for setting metadata*/
	MetadataSetPath = "/metadata/set"

	/*MetadataCreatePath is the path for creating a new metadata*/
	MetadataCreatePath = "/metadata/create"

	/*MetadataDeletePath is the path for deleting a metadata*/
	MetadataDeletePath = "/metadata/delete"

	/*InitUploadPath is the path for uploading files to paid accounts*/
	InitUploadPath = "/init-upload"

	/*UploadPath is the path for uploading files to paid accounts*/
	UploadPath = "/upload"

	/*UploadStatusPath is the path for checking upload status*/
	UploadStatusPath = "/upload-status"

	/*DeletePath is the path for deleting files*/
	DeletePath = "/delete"

	/*DownloadPath is the path for downloading files*/
	DownloadPath = "/download"

	/*StripeCreatePath is the path for creating a stripe payment*/
	StripeCreatePath = "/stripe/create"
)

const MaxRequestSize = utils.MaxMultiPartSize + 1000

var maintenanceError = errors.New("maintenance in progress, currently rejecting writes")

type StatusRes struct {
	Status string `json:"status" example:"status of the request"`
}

type PlanResponse struct {
	Plans utils.PlanResponseType `json:"plans" example:"an object of the plans we offer"`
}

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

	router.GET("/plans", GetPlansHandler())

	return router
}

func returnV1Group(router *gin.Engine) *gin.RouterGroup {
	return router.Group(V1Path)
}

func setupV1Paths(v1Router *gin.RouterGroup) {
	v1Router.POST(AccountsPath, CreateAccountHandler())
	v1Router.POST(AccountDataPath, CheckAccountPaymentStatusHandler())
	v1Router.POST(AccountUpgradeInvoicePath, GetAccountUpgradeInvoiceHandler())
	v1Router.POST(AccountUpgradePath, CheckUpgradeStatusHandler())

	v1Router.POST(MetadataSetPath, UpdateMetadataHandler())
	v1Router.POST(MetadataGetPath, GetMetadataHandler())
	v1Router.POST(MetadataHistoryPath, GetMetadataHistoryHandler())
	v1Router.POST(MetadataCreatePath, CreateMetadataHandler())
	v1Router.POST(MetadataDeletePath, DeleteMetadataHandler())

	v1Router.POST(InitUploadPath, InitFileUploadHandler())
	v1Router.POST(UploadPath, UploadFileHandler())
	v1Router.POST(UploadStatusPath, CheckUploadStatusHandler())

	// File endpoint
	v1Router.POST(DeletePath, DeleteFileHandler())
	v1Router.POST(DownloadPath, DownloadFileHandler())

	// Stripe endpoints
	v1Router.POST(StripeCreatePath, CreateStripePaymentHandler())
}

func setupAdminPaths(router *gin.Engine) {
	g := router.Group(AdminPath, gin.BasicAuth(gin.Accounts{
		utils.Env.AdminUser: utils.Env.AdminPassword,
	}))

	g.GET("/jobrunner/json", jobs.JobJson)

	g.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Load template file location relative to the current working directory
	// Unable to find the file.
	// g.GET("/jobrunner/html", jobs.JobHtml)
	//router.LoadHTMLGlob("../../bamzi/jobrunner/views/Status.html")
}

// GetPlansHandler godoc
// @Summary get the plans we sell
// @Description get the plans we sell
// @Accept  json
// @Produce  json
// @Success 200 {object} routes.PlanResponse
// @Router /plans [get]
/*GetPlansHandler is a handler for getting the plans*/
func GetPlansHandler() gin.HandlerFunc {
	return ginHandlerFunc(getPlans)
}

func getPlans(c *gin.Context) error {
	return OkResponse(c, PlanResponse{
		Plans: utils.Env.Plans,
	})
}
