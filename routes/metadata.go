package routes

import (
	"errors"
	"fmt"
	"time"

	"github.com/dgraph-io/badger"
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
	requestBody
	updateMetadataObject updateMetadataObject
}

type updateMetadataRes struct {
	MetadataKey    string    `json:"metadataKey" binding:"required,len=64" example:"a 64-char hex string created deterministically, will be a key for the metadata of one of your folders"`
	Metadata       string    `json:"metadata" binding:"required" example:"your (updated) account metadata"`
	ExpirationDate time.Time `json:"expirationDate" binding:"required,gte"`
}

type metadataKeyObject struct {
	MetadataKey string `json:"metadataKey" binding:"required,len=64" example:"a 64-char hex string created deterministically, will be a key for the metadata of one of your folders"`
	Timestamp   int64  `json:"timestamp" binding:"required"`
}

type metadataKeyReq struct {
	verification
	requestBody
	metadataKeyObject metadataKeyObject
}

type getMetadataRes struct {
	Metadata       string    `json:"metadata" binding:"exists" example:"your account metadata"`
	ExpirationDate time.Time `json:"expirationDate" binding:"required"`
}

type getMetadataHistoryRes struct {
	Metadata        string    `json:"metadata" binding:"exists" example:"your account metadata"`
	MetadataHistory []string  `json:"metadataHistory" binding:"exists" example:"your account metadata"`
	ExpirationDate  time.Time `json:"expirationDate" binding:"required"`
}

type createMetadataRes struct {
	ExpirationDate time.Time `json:"expirationDate" binding:"required"`
}

var metadataDeletedRes = StatusRes{
	Status: "metadata successfully deleted",
}

func (v *updateMetadataReq) getObjectRef() interface{} {
	return &v.updateMetadataObject
}

const numMetadatasToRetain = 5

func (v *metadataKeyReq) getObjectRef() interface{} {
	return &v.metadataKeyObject
}

// GetMetadataHandler godoc
// @Summary Retrieve account metadata
// @Accept  json
// @Produce  json
// @Param metadataKeyReq body routes.metadataKeyReq true "object for endpoints that only need metadataKey and timestamp"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"metadataKey": "a 64-char hex string created deterministically, will be a key for the metadata of one of your folders",
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

// GetMetadataHistoryHandler godoc
// @Summary Retrieve metadata history
// @Accept  json
// @Produce  json
// @Param metadataKeyReq body routes.metadataKeyReq true "object for endpoints that only need metadataKey and timestamp"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"metadataKey": "a 64-char hex string created deterministically, will be a key for the metadata of one of your folders",
// @description 	"timestamp": 1557346389
// @description }
// @Success 200 {object} routes.getMetadataHistoryRes
// @Failure 404 {string} string "no value found for that key, or account not found"
// @Failure 403 {string} string "subscription expired, or the invoice resonse"
// @Router /api/v1/metadata/history [post]
/*GetMetadataHistoryHandler is a handler for getting the file metadata history*/
func GetMetadataHistoryHandler() gin.HandlerFunc {
	return ginHandlerFunc(getMetadataHistory)
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
// @Failure 404 {string} string "no value found for that key, or account not found"
// @Failure 403 {string} string "subscription expired, or the invoice response"
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
// @Param metadataKeyReq body routes.metadataKeyReq true "object for endpoints that only need metadataKey and timestamp"
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

// DeleteMetadataHandler godoc
// @Summary delete a metadata
// @Accept  json
// @Produce  json
// @Param metadataKeyReq body routes.metadataKeyReq true "object for endpoints that only need metadataKey and timestamp"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"metadataKey": "a 64-char hex string created deterministically, will be a key for the metadata of one of your folders",
// @description 	"timestamp": 1557346389
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 404 {string} string "account not found"
// @Failure 403 {string} string "subscription expired, or the invoice resonse"
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/metadata/delete [post]
/*DeleteMetadataHandler is a handler for deleting a metadata*/
func DeleteMetadataHandler() gin.HandlerFunc {
	return ginHandlerFunc(deleteMetadata)
}

func getMetadata(c *gin.Context) error {
	request := metadataKeyReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	if err := verifyIfPaidWithContext(account, c); err != nil {
		return err
	}

	requestBodyParsed := metadataKeyObject{}

	if err := verifyAndParseStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c); err != nil {
		return err
	}

	permissionHashKey := getPermissionHashKeyForBadger(requestBodyParsed.MetadataKey)
	permissionHashInBadger, _, err := utils.GetValueFromKV(permissionHashKey)

	if err != nil {
		// TODO: Enable this after everyone should have migrated.
		// cannot enable it now since many users already created metadatas without permission hashes
		// being stored
		//
		//return NotFoundResponse(c, err)
	}

	// TODO remove this if block wrapping the other if after everyone should have migrated
	// This is only in effect for a limited time because many users already created metadatas without
	// permission hashes being stored
	if permissionHashInBadger != "" {
		if err := verifyPermissions(request.PublicKey, requestBodyParsed.MetadataKey,
			permissionHashInBadger, c); err != nil {
			return err
		}
	}

	metadata, expirationTime, err := utils.GetValueFromKV(request.metadataKeyObject.MetadataKey)

	if err != nil {
		return NotFoundResponse(c, err)
	}

	return OkResponse(c, getMetadataRes{
		Metadata:       metadata,
		ExpirationDate: expirationTime,
	})
}

func getMetadataHistory(c *gin.Context) error {
	request := metadataKeyReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	if err := verifyIfPaidWithContext(account, c); err != nil {
		return err
	}

	requestBodyParsed := metadataKeyObject{}

	if err := verifyAndParseStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c); err != nil {
		return err
	}

	permissionHashKey := getPermissionHashKeyForBadger(requestBodyParsed.MetadataKey)
	permissionHashInBadger, _, err := utils.GetValueFromKV(permissionHashKey)

	if err != nil {
		return NotFoundResponse(c, err)
	}

	if err := verifyPermissions(request.PublicKey, requestBodyParsed.MetadataKey,
		permissionHashInBadger, c); err != nil {
		return err
	}

	currentMetadata, expirationTime, err := utils.GetValueFromKV(request.metadataKeyObject.MetadataKey)
	if err != nil {
		return NotFoundResponse(c, err)
	}

	metadataHistory, err := getMetadataHistoryWithoutContext(request.metadataKeyObject.MetadataKey)
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, getMetadataHistoryRes{
		Metadata:        currentMetadata,
		MetadataHistory: metadataHistory,
		ExpirationDate:  expirationTime,
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

	if err := verifyIfPaidWithContext(account, c); err != nil {
		return err
	}

	requestBodyParsed := updateMetadataObject{}

	if err := verifyAndParseStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c); err != nil {
		return err
	}

	oldMetadata, _, err := utils.GetValueFromKV(requestBodyParsed.MetadataKey)

	if err != nil {
		return NotFoundResponse(c, err)
	}

	if account.ExpirationDate().Before(time.Now()) {
		return ForbiddenResponse(c, errors.New("subscription expired"))
	}

	permissionHashKey := getPermissionHashKeyForBadger(requestBodyParsed.MetadataKey)
	permissionHashInBadger, _, err := utils.GetValueFromKV(permissionHashKey)

	if err := verifyPermissions(request.PublicKey, requestBodyParsed.MetadataKey,
		permissionHashInBadger, c); err != nil {
		return err
	}
	if err := account.UpdateMetadataSizeInBytes(int64(len(oldMetadata)), int64(len(requestBodyParsed.Metadata))); err != nil {
		return ForbiddenResponse(c, err)
	}

	ttl := time.Until(account.ExpirationDate())

	if err := utils.BatchSet(&utils.KVPairs{
		requestBodyParsed.MetadataKey: requestBodyParsed.Metadata,
		permissionHashKey:             permissionHashInBadger,
	}, ttl); err != nil {
		return InternalErrorResponse(c, err)
	}

	if err := storeMetadataHistory(requestBodyParsed.MetadataKey, oldMetadata, ttl, c); err != nil {
		return err
	}

	return OkResponse(c, updateMetadataRes{
		MetadataKey:    request.updateMetadataObject.MetadataKey,
		Metadata:       request.updateMetadataObject.Metadata,
		ExpirationDate: account.ExpirationDate(),
	})
}

func createMetadata(c *gin.Context) error {
	request := metadataKeyReq{}

	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body: %v", err)
		return BadRequestResponse(c, err)
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	if err := verifyIfPaidWithContext(account, c); err != nil {
		return err
	}

	requestBodyParsed := metadataKeyObject{}

	if err := verifyAndParseStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c); err != nil {
		return err
	}

	ttl := time.Until(account.ExpirationDate())

	permissionHash, err := getPermissionHash(request.PublicKey, requestBodyParsed.MetadataKey, c)
	if err != nil {
		return err
	}

	permissionHashKey := getPermissionHashKeyForBadger(requestBodyParsed.MetadataKey)

	_, _, err = utils.GetValueFromKV(requestBodyParsed.MetadataKey)

	if err == nil {
		return ForbiddenResponse(c, errors.New("that metadata already exists"))
	}

	if err = account.IncrementMetadataCount(); err != nil {
		return ForbiddenResponse(c, err)
	}

	if err = utils.BatchSet(&utils.KVPairs{
		requestBodyParsed.MetadataKey: "",
		permissionHashKey:             permissionHash,
	}, ttl); err != nil {
		account.DecrementMetadataCount()
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, createMetadataRes{
		ExpirationDate: account.ExpirationDate(),
	})
}

func deleteMetadata(c *gin.Context) error {
	request := metadataKeyReq{}

	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body: %v", err)
		return BadRequestResponse(c, err)
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	if err := verifyIfPaidWithContext(account, c); err != nil {
		return err
	}

	requestBodyParsed := metadataKeyObject{}

	if err := verifyAndParseStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c); err != nil {
		return err
	}

	permissionHashKey := getPermissionHashKeyForBadger(requestBodyParsed.MetadataKey)
	permissionHashInBadger, _, err := utils.GetValueFromKV(permissionHashKey)

	if err != nil {
		// TODO: Enable this after everyone should have migrated.
		// cannot enable it now since many users already created metadatas without permission hashes
		// being stored
		//
		//return NotFoundResponse(c, err)
	}

	// TODO remove this if block wrapping the other if after everyone should have migrated
	// This is only in effect for a limited time because many users already created metadatas without
	// permission hashes being stored
	if permissionHashInBadger != "" {
		if err := verifyPermissions(request.PublicKey, requestBodyParsed.MetadataKey,
			permissionHashInBadger, c); err != nil {
			return err
		}
	}

	oldMetadata, _, err := utils.GetValueFromKV(requestBodyParsed.MetadataKey)

	// TODO remove this if block wrapping the other if after everyone should have migrated
	// This is only in effect for a limited time because many users already created metadatas without
	// permission hashes being stored
	if permissionHashInBadger != "" {
		if err := account.RemoveMetadata(int64(len(oldMetadata))); err != nil {
			return InternalErrorResponse(c, err)
		}
	}

	if err = utils.BatchDelete(&utils.KVKeys{
		requestBodyParsed.MetadataKey,
		permissionHashKey,
	}); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, metadataDeletedRes)
}

func storeMetadataHistory(metadataKey string, oldMetadata string, ttl time.Duration, c *gin.Context) error {
	newValue := oldMetadata
	stopOnNextKey := false
	for i := 0; i < numMetadatasToRetain; i++ {
		if stopOnNextKey {
			break
		}
		badgerKey := getVersionKeyForBadger(metadataKey, i)
		oldValue, _, err := utils.GetValueFromKV(badgerKey)
		if err := utils.BatchSet(&utils.KVPairs{
			badgerKey: newValue,
		}, ttl); err != nil {
			return InternalErrorResponse(c, err)
		}
		if err == badger.ErrKeyNotFound {
			stopOnNextKey = true
		}
		newValue = oldValue
	}
	return nil
}

func getMetadataHistoryWithoutContext(metadataKey string) ([]string, error) {
	metadataHistory := []string{}
	for i := 0; i < numMetadatasToRetain; i++ {
		oldMetadata, _, err := utils.GetValueFromKV(getVersionKeyForBadger(metadataKey, i))
		if err == badger.ErrKeyNotFound {
			break
		}
		if err != nil {
			return metadataHistory, err
		}
		metadataHistory = append(metadataHistory, oldMetadata)
	}
	return metadataHistory, nil
}

func getCurrentMetadata() {
	// TODO:  After everyone has migrated, consolidate the first parts of getMetadata and
	// getMetadataHistory since they are very similar except for the exceptions for accounts
	// that haven't been migrated
}
