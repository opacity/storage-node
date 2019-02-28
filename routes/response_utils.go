package routes

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

func InternalError(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, err.Error())
}

func BadRequest(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
}

func ServiceUnavailable(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusServiceUnavailable, err.Error())
}

func Forbidden(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusForbidden, err.Error())
}

func NotFound(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusNotFound, err.Error())
}

func OkResponse(c *gin.Context, response interface{}) {
	if err := utils.Validator.Struct(response); err != nil {
		err = fmt.Errorf("could not create a valid response:  %v", err)
		BadRequest(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
}
