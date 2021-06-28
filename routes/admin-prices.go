package routes

import "github.com/gin-gonic/gin"

func AdminPricesHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminPrices)
}

func adminPrices(c *gin.Context) error {
	defer c.Request.Body.Close()

	return OkResponse(c, StatusRes{Status: "prices were updated"})
}
