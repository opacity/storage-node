package routes

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

// must be sorted alphabetically for JSON marshaling/stringifying
type updateMetadataObject struct {
	Metadata    string `json:"metadata" binding:"required" example:"your (updated) account metadata"`
	MetadataKey string `json:"metadataKey" binding:"required,len=64" example:"a 64-char hex string created deterministically from your account handle or private key"`
	Timestamp   int64  `json:"timestamp" binding:"required"`
}

type updateMetadataReq struct {
	verification
	requestBody
	updateMetadataObject updateMetadataObject
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
	requestBody
	getMetadataObject getMetadataObject
}

type getMetadataRes struct {
	Metadata       string    `json:"metadata" binding:"exists" example:"your account metadata"`
	ExpirationDate time.Time `json:"expirationDate" binding:"required"`
}

func (v *updateMetadataReq) getObjectRef() interface{} {
	return &v.updateMetadataObject
}

func (v *getMetadataReq) getObjectRef() interface{} {
	return &v.getMetadataObject
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
// @Failure 404 {string} string "no value found for that key, or account not found"
// @Failure 403 {string} string "subscription expired, or the invoice resonse"
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
// @description 	"metadataKey": "a 64-char hex string created deterministically from your account handle or private key",
// @description 	"metadata": "your (updated) account metadata",
// @description 	"timestamp": 1557346389
// @description }
// @Success 200 {object} routes.updateMetadataRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 404 {string} string "no value found for that key, or account not found"
// @Failure 403 {string} string "subscription expired, or the invoice resonse"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/metadata/set [post]
/*UpdateMetadataHandler is a handler for updating the file metadata*/
func UpdateMetadataHandler() gin.HandlerFunc {
	return ginHandlerFunc(setMetadata)
}

func getMetadata(c *gin.Context) error {
	request := getMetadataReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	if err := verifyIfPaid(account, c); err != nil {
		return err
	}

	metadata, expirationTime, err := utils.GetValueFromKV(request.getMetadataObject.MetadataKey)
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

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	if err := verifyIfPaid(account, c); err != nil {
		return err
	}

	metadataKey := request.updateMetadataObject.MetadataKey
	if _, _, err := utils.GetValueFromKV(metadataKey); err != nil {
		return NotFoundResponse(c, err)
	}

	if account.ExpirationDate().Before(time.Now()) {
		return ForbiddenResponse(c, errors.New("subscription expired"))
	}

	ttl := time.Until(account.ExpirationDate())

	if err := utils.BatchSet(&utils.KVPairs{metadataKey: request.updateMetadataObject.Metadata}, ttl); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, updateMetadataRes{
		MetadataKey:    metadataKey,
		Metadata:       request.updateMetadataObject.Metadata,
		ExpirationDate: account.ExpirationDate(),
	})
}
