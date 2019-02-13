package routes

import (
	"net/http"

	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

/*CreateRoutes creates our application's routes*/
func CreateRoutes() {
	router := returnEngine()
	// Test app is running
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Gin app is running")
	})

	v2 := returnV2Group(router)

	setupV2Paths(v2)

	// Listen and Serve
	router.Run(":" + os.Getenv("PORT"))
}

func returnEngine() *gin.Engine {
	router := gin.Default()
	config := cors.DefaultConfig()

	// TODO:  update to only allow our frontend and localhost
	config.AllowAllOrigins = true
	router.Use(cors.New(config))

	// Test app is running
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Storage node is running")
	})

	return router
}

func returnV2Group(router *gin.Engine) *gin.RouterGroup {
	return router.Group("/v2")
}

func setupV2Paths(v2Router *gin.RouterGroup) {
	v2Router.POST("/new-subscription", func(c *gin.Context) {
		c.JSON(http.StatusOK, "stub for creating a new subscription")
	})

	v2Router.POST("/trial-upload", func(c *gin.Context) {
		c.JSON(http.StatusOK, "stub for doing a trial upload")
	})

	v2Router.POST("/new-upload", func(c *gin.Context) {
		c.JSON(http.StatusOK, "stub for uploading a file with an existing subscription")
	})
}
