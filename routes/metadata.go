package routes

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

type updateMetadataReq struct {
	MetadataKey string `json:"metadataKey" binding:"required,len=64"`
	Metadata    string `json:"metadata" binding:"required"`
}

type updateMetadataRes struct {
	MetadataKey    string    `json:"metadataKey" binding:"required,len=64"`
	Metadata       string    `json:"metadata" binding:"required"`
	ExpirationDate time.Time `json:"expirationDate" binding:"required,gte"`
}

type getMetadataRes struct {
	Metadata       string    `json:"metadata"`
	ExpirationDate time.Time `json:"expirationDate" binding:"required,gte"`
}

/*GetMetadataHandler is a handler for getting the file metadata*/
func GetMetadataHandler() gin.HandlerFunc {
	return ginHandlerFunc(getMetadata)
}

/*GetMetadataHandler is a handler for updating the file metadata*/
func UpdateMetadataHandler() gin.HandlerFunc {
	return ginHandlerFunc(setMetadata)
}

func getMetadata(c *gin.Context) {
	metadataKey := c.Param("metadataKey")

	metadata, expirationTime, err := utils.GetValueFromKV(metadataKey)
	if err != nil {
		NotFoundResponse(c, err)
		return
	}
	OkResponse(c, getMetadataRes{
		Metadata:       metadata,
		ExpirationDate: expirationTime,
	})
}

func setMetadata(c *gin.Context) {
	request := updateMetadataReq{}

	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		BadRequestResponse(c, err)
		return
	}

	_, expirationTime, err := utils.GetValueFromKV(request.MetadataKey)

	if err != nil {
		NotFoundResponse(c, err)
		return
	}

	if expirationTime.Before(time.Now()) {
		ForbiddenResponse(c, errors.New("subscription expired"))
		return
	}

	ttl := time.Until(expirationTime)

	if err := utils.BatchSet(&utils.KVPairs{request.MetadataKey: request.Metadata}, ttl); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	OkResponse(c, updateMetadataRes{
		MetadataKey:    request.MetadataKey,
		Metadata:       request.Metadata,
		ExpirationDate: expirationTime,
	})
}
