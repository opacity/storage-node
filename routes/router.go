package routes

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var uptime time.Time

/*CreateRoutes creates our application's routes*/
func CreateRoutes() {
	uptime = time.Now()

	router := returnEngine()

	v2 := returnV2Group(router)

	setupV2Paths(v2)

	// Listen and Serve
	err := router.Run(":" + os.Getenv("PORT"))
	fmt.Printf("Error in running %s", err)
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
