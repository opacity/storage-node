package routes

import (
	"errors"
	"fmt"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

// must be sorted alphabetically for JSON marshaling/stringifying
type updateMetadataObject struct {
	Metadata    string `json:"metadata" binding:"required,len=64" example:"your (updated) account metadata"`
	MetadataKey string `json:"metadataKey" binding:"required,len=64" example:"a 64-char hex string created deterministically from your account handle or private key"`
	Timestamp   int64  `json:"timestamp" binding:"required"`
}

type updateMetadataReq struct {
	verification
	Metadata updateMetadataObject `json:"metadata" binding:"required"`
}

type updateMetadataRes struct {
	MetadataKey    string    `json:"metadataKey" binding:"required,len=64" example:"a 64-char hex string created deterministically from your account handle or private key"`
	Metadata       string    `json:"metadata" binding:"required" example:"your (updated) account metadata"`
	ExpirationDate time.Time `json:"expirationDate" binding:"required,gte"`
}

type getMetadataRes struct {
	Metadata       string    `json:"metadata" binding:"exists" example:"your account metadata"`
	ExpirationDate time.Time `json:"expirationDate" binding:"required"`
}

// GetMetadataHandler godoc
// @Summary retrieve account metadata
// @Description retrieve account metadata
// @Accept  json
// @Produce  json
// @Param metadataKey path string true "your metadata key which is 64-char hex string created deterministically from your account handle or private key"
// @Success 200 {object} routes.getMetadataRes
// @Failure 404 {string} string "no value found for that key"
// @Router /api/v1/metadata/putMetadataKeyHere [get]
/*GetMetadataHandler is a handler for getting the file metadata*/
func GetMetadataHandler() gin.HandlerFunc {
	return ginHandlerFunc(getMetadata)
}

// UpdateMetadataHandler godoc
// @Summary update metadata
// @Description update metadata
// @Accept  json
// @Produce  json
// @Param updateMetadataReq body routes.updateMetadataReq true "update metadata object"
// @Success 200 {object} routes.updateMetadataRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 404 {string} string "no value found for that key"
// @Failure 403 {string} string "subscription expired"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/metadata [post]
/*UpdateMetadataHandler is a handler for updating the file metadata*/
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
		err = fmt.Errorf("bad request, unable to parse request body: %v", err)
		BadRequestResponse(c, err)
		return
	}

	if err := verifyRequest(request.Metadata, request.Address, request.Signature, c); err != nil {
		return
	}

	_, expirationTime, err := utils.GetValueFromKV(request.Metadata.MetadataKey)

	if err != nil {
		NotFoundResponse(c, err)
		return
	}

	if expirationTime.Before(time.Now()) {
		ForbiddenResponse(c, errors.New("subscription expired"))
		return
	}

	ttl := time.Until(expirationTime)

	if err := utils.BatchSet(&utils.KVPairs{request.Metadata.MetadataKey: request.Metadata.Metadata}, ttl); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	OkResponse(c, updateMetadataRes{
		MetadataKey:    request.Metadata.MetadataKey,
		Metadata:       request.Metadata.Metadata,
		ExpirationDate: expirationTime,
	})
}
