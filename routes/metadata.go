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
	RequestBody string `json:"requestBody" binding:"required" example:"should produce routes.updateMetadataObject, see description for example"`
}

type updateMetadataRes struct {
	MetadataKey    string    `json:"metadataKey" binding:"required,len=64" example:"a 64-char hex string created deterministically from your account handle or private key"`
	Metadata       string    `json:"metadata" binding:"required" example:"your (updated) account metadata"`
	ExpirationDate time.Time `json:"expirationDate" binding:"required,gte"`
}

type getMetadataObject struct {
	MetadataKey string `json:"metadataKey" binding:"required,len=64" example:"a 64-char hex string created deterministically from your account handle or private key"`
	Timestamp   int64  `json:"timestamp" binding:"required"`
}

type getMetadataReq struct {
	verification
	RequestBody string `json:"requestBody" binding:"required" example:"should produce routes.getMetadataObject, see description for example"`
}

type getMetadataRes struct {
	Metadata       string    `json:"metadata" binding:"exists" example:"your account metadata"`
	ExpirationDate time.Time `json:"expirationDate" binding:"required"`
}

// GetMetadataHandler godoc
// @Summary Retrieve account metadata
// @Accept  json
// @Produce  json
// @Param getMetadataReq body routes.getMetadataReq true "get metadata object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"metadataKey": "a 64-char hex string created deterministically from your account handle or private key",
// @description 	"timestamp": 1557346389
// @description }
// @Success 200 {object} routes.getMetadataRes
// @Failure 404 {string} string "no value found for that key"
// @Router /api/v1/metadata [get]
/*GetMetadataHandler is a handler for getting the file metadata*/
func GetMetadataHandler() gin.HandlerFunc {
	return ginHandlerFunc(getMetadata)
}

// UpdateMetadataHandler godoc
// @Summary Update metadata
// @Accept  json
// @Produce  json
// @Param updateMetadataReq body routes.updateMetadataReq true "update metadata object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"metadataKey": "a 64-char hex string created deterministically from your account handle or private key",
// @description 	"metadata": "your (updated) account metadata",
// @description 	"timestamp": 1557346389
// @description }
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
	request := getMetadataReq{}

	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body: %v", err)
		BadRequestResponse(c, err)
		return
	}

	requestBodyParsed := getMetadataObject{}

	if err := verifyAndParseStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c); err != nil {
		return
	}

	metadata, expirationTime, err := utils.GetValueFromKV(requestBodyParsed.MetadataKey)
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

	requestBodyParsed := updateMetadataObject{}

	if err := verifyAndParseStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c); err != nil {
		return
	}

	_, expirationTime, err := utils.GetValueFromKV(requestBodyParsed.MetadataKey)

	if err != nil {
		NotFoundResponse(c, err)
		return
	}

	if expirationTime.Before(time.Now()) {
		ForbiddenResponse(c, errors.New("subscription expired"))
		return
	}

	ttl := time.Until(expirationTime)

	if err := utils.BatchSet(&utils.KVPairs{requestBodyParsed.MetadataKey: requestBodyParsed.Metadata}, ttl); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	OkResponse(c, updateMetadataRes{
		MetadataKey:    requestBodyParsed.MetadataKey,
		Metadata:       requestBodyParsed.Metadata,
		ExpirationDate: expirationTime,
	})
}
