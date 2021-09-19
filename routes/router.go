package routes

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "github.com/opacity/storage-node/docs"

	sentrygin "github.com/getsentry/sentry-go/gin"
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

	/*AccountRenewInvoicePath is the path for getting an invoice to renew an account*/
	AccountRenewInvoicePath = "/renew/invoice"

	/*AccountRenewPath is the path for checking the renew status of an account*/
	AccountRenewPath = "/renew"

	/*AccountUpdateApiVersion is the path for updating the API version to v2 */
	AccountUpdateApiVersion = "/account/updateApiVersion"

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

	/*DeletePath is the path for deleting files*/
	DeletePath = "/delete"

	/*DownloadPath is the path for downloading files*/
	DownloadPath = "/download"

	/*StripeCreatePath is the path for creating a stripe payment*/
	StripeCreatePath = "/stripe/create"
)

const (
	/*V2Path is a router group for the v1 version of storage node*/
	V2Path = "/api/v2"

	/*AccountUpgradeV2InvoicePath is the path for getting an invoice to upgrade an account*/
	AccountUpgradeV2InvoicePath = "/upgrade/invoice"

	/*AccountUpgradeV2Path is the path for checking the upgrade status of an account*/
	AccountUpgradeV2Path = "/upgrade"

	/*AccountRenewV2InvoicePath is the path for getting an invoice to renew an account*/
	AccountRenewV2InvoicePath = "/renew/invoice"

	/*AccountRenewV2Path is the path for checking the renew status of an account*/
	AccountRenewV2Path = "/renew"

	/*DownloadV2Path is the path for downloading files*/
	DownloadV2Path = "/download/private"

	/*DownloadPublicV2Path is the path for downloading public files*/
	DownloadPublicV2Path = "/download/public"

	/*MetadataV2GetPath is the path for getting metadata*/
	MetadataV2GetPath = "/metadata/get"

	/*MetadataV2GetPublicPath is the path for getting metadata*/
	MetadataV2GetPublicPath = "/metadata/get-public"

	/*MetadataV2AddPath is the path for setting metadata*/
	MetadataV2AddPath = "/metadata/add"

	/*MetadataV2DeletePath is the path for deleting a metadata*/
	MetadataV2DeletePath = "/metadata/delete"

	/*MetadataMultipleV2DeletePath is the path for deleting multiple metadata*/
	MetadataMultipleV2DeletePath = "/metadata/delete-multiple"

	/*InitUploadPublicPath is the path for initiating the upload of files for public sharing*/
	InitUploadPublicPath = "/init-upload-public"

	/*UploadPublicPath is the path for uploading files for public sharing*/
	UploadPublicPath = "/upload-public"

	/*UploadStatusPath is the path for checking upload status*/
	UploadStatusPath = "/upload-status"

	/*UploadStatusPublicPath is the path for checking upload status*/
	UploadStatusPublicPath = "/upload-status-public"

	/*PublicSharePathPrefix is the base path public shared files*/
	PublicSharePathPrefix = "public-share"

	/*PrivateToPublicConvertPath is the path for converting a private file to a public shared one*/
	PrivateToPublicConvertPath = "/convert"

	/*CreateShortLinkPath is the path for creating a shortlink of a public shared file */
	CreateShortLinkPath = "/shortlink"

	/*PublicShareShortlinkPath is the path for getting the shortlink of a public shared files*/
	PublicShareShortlinkPath = "/:shortlink"

	/*PublicShareViewsCountPath is the path for getting the shortlink of a public shared file*/
	PublicShareViewsCountPath = "/views-count"

	/*PublicShareRevokePath is the path for revoking the share of a public file*/
	PublicShareRevokePath = "/revoke"

	/*DeletePath is the path for deleting files, allowing multiple deletions*/
	DeleteV2Path = "/delete"
)

const MaxRequestSize = utils.MaxMultiPartSize + 1000

var maintenanceError = errors.New("maintenance in progress, currently rejecting writes")

// StatusRes ...
type StatusRes struct {
	Status string `json:"status" example:"status of the request"`
}

// PlanResponse ...
type PlanResponse struct {
	Plans utils.PlanResponseType `json:"plans"`
}

func init() {
}

/*CreateRoutes creates our application's routes*/
func CreateRoutes() {
	uptime = time.Now()

	router := returnEngine()

	setupV1Paths(returnV1Group(router))
	setupV2Paths(returnV2Group(router))
	setupAdminPaths(router)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Listen and Serve
	err := router.Run(":" + os.Getenv("PORT"))
	utils.LogIfError(err, map[string]interface{}{"error": err})
}

func returnEngine() *gin.Engine {
	router := gin.Default()
	if utils.Env.GoEnv == "production" || utils.Env.GoEnv == "dev2" {
		router.Use(sentrygin.New(sentrygin.Options{
			Repanic: true,
		}))
	}
	config := cors.DefaultConfig()

	// TODO:  update to only allow our frontend and localhost
	config.AllowAllOrigins = true
	config.AllowHeaders = append(config.AllowHeaders, "sentry-trace")
	router.Use(cors.New(config))

	// Test app is running
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]interface{}{
			"message": "Storage node is running",
			"uptime":  fmt.Sprintf("%v", time.Since(uptime)),
			"version": utils.Env.Version,
		})
	})

	router.GET("/plans", GetPlansHandler())

	return router
}

func returnV1Group(router *gin.Engine) *gin.RouterGroup {
	return router.Group(V1Path)
}

func returnV2Group(router *gin.Engine) *gin.RouterGroup {
	return router.Group(V2Path)
}

func setupV1Paths(v1Router *gin.RouterGroup) {
	v1Router.POST(AccountsPath, CreateAccountHandler())
	v1Router.POST(AccountDataPath, CheckAccountPaymentStatusHandler())

	v1Router.POST(AccountUpgradeInvoicePath, GetAccountUpgradeInvoiceHandler())
	v1Router.POST(AccountUpgradePath, CheckUpgradeStatusHandler())

	v1Router.POST(AccountRenewInvoicePath, GetAccountRenewalInvoiceHandler())
	v1Router.POST(AccountRenewPath, CheckRenewalStatusHandler())

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

func setupV2Paths(v2Router *gin.RouterGroup) {
	v2Router.POST(AccountRenewV2InvoicePath, GetAccountRenewalV2InvoiceHandler())
	v2Router.POST(AccountRenewV2Path, CheckRenewalV2StatusHandler())

	v2Router.POST(AccountUpgradeV2InvoicePath, GetAccountUpgradeV2InvoiceHandler())
	v2Router.POST(AccountUpgradeV2Path, CheckUpgradeV2StatusHandler())

	v2Router.POST(MetadataV2AddPath, UpdateMetadataV2Handler())
	v2Router.POST(MetadataV2GetPath, GetMetadataV2Handler())
	v2Router.POST(MetadataV2GetPublicPath, GetMetadataV2PublicHandler())
	v2Router.POST(MetadataV2DeletePath, DeleteMetadataV2Handler())
	v2Router.POST(MetadataMultipleV2DeletePath, DeleteMetadataMultipleV2Handler())

	v2Router.POST(InitUploadPublicPath, InitFileUploadPublicHandler())
	v2Router.POST(UploadPublicPath, UploadFilePublicHandler())
	v2Router.POST(UploadStatusPublicPath, CheckUploadStatusPublicHandler())

	v2Router.POST(AccountUpdateApiVersion, AccountUpdateApiVersionHandler())

	v2Router.POST(DownloadV2Path, DownloadFileHandler())
	v2Router.POST(DownloadPublicV2Path, DownloadPublicFileHandler())

	publicShareRouterGroup := v2Router.Group(PublicSharePathPrefix)
	publicShareRouterGroup.GET(PublicShareShortlinkPath, ShortlinkFileHandler())
	publicShareRouterGroup.POST(PrivateToPublicConvertPath, PrivateToPublicConvertHandler())
	publicShareRouterGroup.POST(CreateShortLinkPath, CreateShortlinkHandler())
	publicShareRouterGroup.POST(PublicShareViewsCountPath, ViewsCountHandler())
	publicShareRouterGroup.POST(PublicShareRevokePath, RevokePublicShareHandler())

	v2Router.POST(DeleteV2Path, DeleteFilesHandler())
}

func setupAdminPaths(router *gin.Engine) {
	router.LoadHTMLGlob("templates/*")

	g := router.Group(AdminPath, gin.BasicAuth(gin.Accounts{
		utils.Env.AdminUser: utils.Env.AdminPassword,
	}))

	g.GET("/jobrunner/json", jobs.JobJson)

	g.GET("/metrics", gin.WrapH(promhttp.Handler()))

	g.GET("/delete", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin-delete.tmpl", gin.H{
			"title": "Delete File",
		})
	})
	g.POST("/delete", AdminDeleteFileHandler())

	setupAdminPlansPaths(g)

	// Load template file location relative to the current working directory
	// Unable to find the file.
	// g.GET("/jobrunner/html", jobs.JobHtml)
	//router.LoadHTMLGlob("../../bamzi/jobrunner/views/Status.html")
}

func setupAdminPlansPaths(adminGroup *gin.RouterGroup) {
	plansGroup := adminGroup.Group("/plans")

	plansGroup.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "plans-list.tmpl", gin.H{
			"title": "Change plans",
			"plans": utils.Env.Plans,
		})
	})
	plansGroup.GET("/edit/:plan", AdminPlansGetHandler())
	plansGroup.GET("/confirm-remove/:plan", AdminPlansRemoveConfirmHandler())
	plansGroup.POST("/remove/:plan", AdminPlansRemoveHandler())
	plansGroup.POST("/", AdminPlansChangeHandler())
	plansGroup.GET("/add", func(c *gin.Context) {
		c.HTML(http.StatusOK, "plan-add.tmpl", gin.H{
			"title": "Add plan",
		})
	})
	plansGroup.POST("/add", AdminPlansAddHandler())
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
