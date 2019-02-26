package routes

import (
	"net/http"

	"time"

	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

type updateMetadataReq struct {
	MetadataKey string `json:"metadataKey" binding:"required,len=64"`
	Metadata    string `json:"Metadata" binding:"required"`
}

/*GetMetadataHandler is a handler for getting the file metadata*/
func GetMetadataHandler() gin.HandlerFunc {
	return gin.HandlerFunc(getMetadata)
}

/*GetMetadataHandler is a handler for updating the file metadata*/
func UpdateMetadataHandler() gin.HandlerFunc {
	return gin.HandlerFunc(setMetadata)
}

func getMetadata(c *gin.Context) {
	metadataKey := c.Param("metadataKey")

	metadata, _, err := utils.GetValueFromKV(metadataKey)
	if err != nil {
		c.JSON(http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, metadata)
}

func setMetadata(c *gin.Context) {
	request := updateMetadataReq{}

	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
		return
	}

	_, expirationTime, err := utils.GetValueFromKV(request.MetadataKey)
	if err != nil {
		c.JSON(http.StatusNotFound, err.Error())
		return
	}

	if expirationTime.Before(time.Now()) {
		c.JSON(http.StatusForbidden, "subscription expired")
		return
	}

	ttl := time.Until(expirationTime)

	if err := utils.BatchSet(&utils.KVPairs{request.MetadataKey: request.Metadata}, ttl); err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, err.Error())
		return
	}

	c.JSON(http.StatusOK, request)
}
