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
	Metadata    string `json:"metadata" binding:"required" example:"your (updated) account metadata"`
	MetadataKey string `json:"metadataKey" binding:"required,len=64" example:"a 64-char hex string created deterministically, will be a key for the metadata of one of your folders"`
	Timestamp   int64  `json:"timestamp" binding:"required"`
}

type updateMetadataReq struct {
	verification
	RequestBody string `json:"requestBody" binding:"required" example:"should produce routes.updateMetadataObject, see description for example"`
}

type updateMetadataRes struct {
	MetadataKey    string    `json:"metadataKey" binding:"required,len=64" example:"a 64-char hex string created deterministically, will be a key for the metadata of one of your folders"`
	Metadata       string    `json:"metadata" binding:"required" example:"your (updated) account metadata"`
	ExpirationDate time.Time `json:"expirationDate" binding:"required,gte"`
}

type getOrCreateMetadataObject struct {
	MetadataKey string `json:"metadataKey" binding:"required,len=64" example:"a 64-char hex string created deterministically, will be a key for the metadata of one of your folders"`
	Timestamp   int64  `json:"timestamp" binding:"required"`
}

type getOrCreateMetadataReq struct {
	verification
	RequestBody string `json:"requestBody" binding:"required" example:"should produce routes.getOrCreateMetadataObject, see description for example"`
}

type getMetadataRes struct {
	Metadata       string    `json:"metadata" binding:"exists" example:"your account metadata"`
	ExpirationDate time.Time `json:"expirationDate" binding:"required"`
}

type createMetadataRes struct {
	ExpirationDate time.Time `json:"expirationDate" binding:"required"`
}

// GetMetadataHandler godoc
// @Summary Retrieve account metadata
// @Accept  json
// @Produce  json
// @Param getOrCreateMetadataReq body routes.getOrCreateMetadataReq true "get or create metadata object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"metadataKey": "a 64-char hex string created deterministically, will be a key for the metadata of one of your folders",
// @description 	"timestamp": 1557346389
// @description }
// @Success 200 {object} routes.getMetadataRes
// @Failure 404 {string} string "no value found for that key"
// @Router /api/v1/metadata/get [post]
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
// @description 	"metadataKey": "a 64-char hex string created deterministically, will be a key for the metadata of one of your folders",
// @description 	"metadata": "your (updated) account metadata",
// @description 	"timestamp": 1557346389
// @description }
// @Success 200 {object} routes.updateMetadataRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 404 {string} string "no value found for that key"
// @Failure 403 {string} string "subscription expired"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/metadata/set [post]
/*UpdateMetadataHandler is a handler for updating the file metadata*/
func UpdateMetadataHandler() gin.HandlerFunc {
	return ginHandlerFunc(setMetadata)
}

// CreateMetadataHandler godoc
// @Summary create a new metadata
// @Accept  json
// @Produce  json
// @Param getOrCreateMetadataReq body routes.getOrCreateMetadataReq true "get or create metadata object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"metadataKey": "a 64-char hex string created deterministically, will be a key for the metadata of one of your folders",
// @description 	"timestamp": 1557346389
// @description }
// @Success 200 {object} routes.createMetadataRes
// @Failure 404 {string} string "account not found"
// @Failure 403 {string} string "subscription expired, or the invoice resonse"
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/metadata/create [post]
/*CreateMetadataHandler is a handler for creating a metadata*/
func CreateMetadataHandler() gin.HandlerFunc {
	return ginHandlerFunc(createMetadata)
}

func getMetadata(c *gin.Context) error {
	request := getOrCreateMetadataReq{}

	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body: %v", err)
		return BadRequestResponse(c, err)
	}

	requestBodyParsed := getOrCreateMetadataObject{}

	if err := verifyAndParseStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c); err != nil {
		return err
	}

	metadata, expirationTime, err := utils.GetValueFromKV(requestBodyParsed.MetadataKey)
	if err != nil {
		return NotFoundResponse(c, err)
	}

	return OkResponse(c, getMetadataRes{
		Metadata:       metadata,
		ExpirationDate: expirationTime,
	})
}

func setMetadata(c *gin.Context) error {
	request := updateMetadataReq{}

	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body: %v", err)
		return BadRequestResponse(c, err)
	}

	requestBodyParsed := updateMetadataObject{}

	if err := verifyAndParseStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c); err != nil {
		return err
	}

	_, expirationTime, err := utils.GetValueFromKV(requestBodyParsed.MetadataKey)

	if err != nil {
		return NotFoundResponse(c, err)
	}

	if expirationTime.Before(time.Now()) {
		return ForbiddenResponse(c, errors.New("subscription expired"))
	}

	ttl := time.Until(expirationTime)

	if err := utils.BatchSet(&utils.KVPairs{requestBodyParsed.MetadataKey: requestBodyParsed.Metadata}, ttl); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, updateMetadataRes{
		MetadataKey:    requestBodyParsed.MetadataKey,
		Metadata:       requestBodyParsed.Metadata,
		ExpirationDate: expirationTime,
	})
}

func createMetadata(c *gin.Context) error {
	request := getOrCreateMetadataReq{}

	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body: %v", err)
		return BadRequestResponse(c, err)
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	if err := verifyIfPaid(account, c); err != nil {
		return err
	}

	requestBodyParsed := getOrCreateMetadataObject{}

	if err := verifyAndParseStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c); err != nil {
		return err
	}

	ttl := time.Until(account.ExpirationDate())

	if err = utils.BatchSet(&utils.KVPairs{requestBodyParsed.MetadataKey: ""}, ttl); err != nil {
		InternalErrorResponse(c, err)
		return err
	}

	return OkResponse(c, createMetadataRes{
		ExpirationDate: account.ExpirationDate(),
	})
}
