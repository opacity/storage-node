package routes

import "github.com/gin-gonic/gin"

func AdminSiaAllowanceGetHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminSiaAllowanceGet)
}

func AdminSiaAllowanceChangeHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminSiaAllowanceChange)
}

func adminSiaAllowanceGet(c *gin.Context) error {

	return nil
}

func adminSiaAllowanceChange(c *gin.Context) error {

	return nil
}
