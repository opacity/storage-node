package routes

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

func InternalErrorResponse(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, err.Error())
}

func BadRequestResponse(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
}

func ServiceUnavailableResponse(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusServiceUnavailable, err.Error())
}

func AccountNotFoundResponse(c *gin.Context, id string) {
	c.AbortWithStatusJSON(http.StatusNotFound, fmt.Sprintf("no account with that id: %s", id))
}

func Forbidden(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusForbidden, err.Error())
}

func NotFoundResponse(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusNotFound, err.Error())
}

func OkResponse(c *gin.Context, response interface{}) {
	if err := utils.Validator.Struct(response); err != nil {
		err = fmt.Errorf("could not create a valid response:  %v", err)
		BadRequestResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
}
