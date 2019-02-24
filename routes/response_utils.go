package routes

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

func BadRequest(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
}

func ServiceUnavailable(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusServiceUnavailable, err.Error())
}

func AccountNotFound(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusNotFound, "no account with that id")
}

func OkResponse(c *gin.Context, response interface{}) {
	if err := utils.Validator.Struct(response); err != nil {
		err = fmt.Errorf("could not create a valid response:  %v", err)
		BadRequest(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
}
