package routes

import (
	"encoding/json"
	"errors"
	"fmt"

	"time"

	"encoding/hex"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

// must be sorted alphabetically for JSON marshaling/stringifying
type updateMetadataObject struct {
	Metadata    string `json:"metadata" binding:"required,len=64"`
	MetadataKey string `json:"metadataKey" binding:"required,len=64"`
	Timestamp   int64  `json:"timestamp" binding:"required"`
}

type updateMetadataReq struct {
	// signature without 0x prefix is broken into
	// V: sig[0:63]
	// R: sig[64:127]
	// S: sig[128:129]
	Signature string               `json:"signature" binding:"required,len=130"`
	Address   string               `json:"address" binding:"required,len=42"`
	Metadata  updateMetadataObject `json:"metadata" binding:"required"`
}

type updateMetadataRes struct {
	MetadataKey    string    `json:"metadataKey" binding:"required,len=64"`
	Metadata       string    `json:"metadata" binding:"required"`
	ExpirationDate time.Time `json:"expirationDate" binding:"required,gte"`
}

type getMetadataRes struct {
	Metadata       string    `json:"metadata" binding:"required,len=64"`
	ExpirationDate time.Time `json:"expirationDate" binding:"required,gte"`
}

/*GetMetadataHandler is a handler for getting the file metadata*/
func GetMetadataHandler() gin.HandlerFunc {
	return ginHandlerFunc(getMetadata)
}

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

	metadataJSON, err := json.Marshal(request.Metadata)
	if err != nil {
		err = fmt.Errorf("bad request, unable to parse metadata body: %v", err)
		BadRequestResponse(c, err)
		return
	}

	hash := utils.Hash(metadataJSON)
	verified, err := utils.VerifyFromStrings(request.Address, hex.EncodeToString(hash),
		request.Signature)
	if err != nil {
		BadRequestResponse(c, errors.New("error verifying signature"))
		return
	}

	if verified != true {
		ForbiddenResponse(c, errors.New("signature did not match"))
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
