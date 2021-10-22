package routes

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/dag"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

const metadataIncorrectKeyLength = "bad request, incorrect key length"
const MetadataExpirationOffset = 24 * time.Hour * 60

// must be sorted alphabetically for JSON marshaling/stringifying
type updateMetadataV2BaseObject struct {
	IsPublic         bool     `json:"isPublic"`
	MetadataV2Edges  []string `json:"metadataV2Edges" validate:"required,dive,base64url,len=12" example:"the edges to add to your account metadataV2 encoded to base64url"`
	MetadataV2Key    string   `json:"metadataV2Key" validate:"required,base64url,len=44" example:"public key for the metadataV2 encoded to base64url"`
	MetadataV2Sig    string   `json:"metadataV2Sig" validate:"required,base64url,len=88" example:"a signature encoded to base64url confirming the metadata change, the publickey will be a key for the metadataV2"`
	MetadataV2Vertex string   `json:"metadataV2Vertex" validate:"required,base64url" example:"the vertex to add to your account metadataV2 encoded to base64url"`
}

type updateMetadataV2Object struct {
	updateMetadataV2BaseObject
	Timestamp int64 `json:"timestamp" validate:"required"`
}

type updateMetadataMultipleV2Object struct {
	Metadatas []updateMetadataV2BaseObject `json:"metadatas" validate:"required"`
	Timestamp int64                        `json:"timestamp" validate:"required"`
}

type updateMetadataV2Req struct {
	verification
	requestBody
	updateMetadataV2Object updateMetadataV2Object
}

type updateMetadataMultipleV2Req struct {
	verification
	requestBody
	updateMetadataMultipleV2Object updateMetadataMultipleV2Object
}

type updateMetadataV2ResBase struct {
	MetadataV2Key string `json:"metadataV2Key" validate:"required,base64url,len=44" example:"public key for the metadataV2 encoded to base64url"`
	MetadataV2    string `json:"metadataV2" validate:"required,base64url" example:"your (updated) account metadataV2"`
}

type updateMetadataV2Res struct {
	updateMetadataV2ResBase
	ExpirationDate time.Time `json:"expirationDate" validate:"required,gte"`
}

type updateMetadataMultipleV2Res struct {
	Metadatas      []updateMetadataV2ResBase `json:"metadatas" validate:"required"`
	ExpirationDate time.Time                 `json:"expirationDate" validate:"required,gte"`
}

type metadataV2KeyObject struct {
	MetadataV2Key string `json:"metadataV2Key" validate:"required,base64url,len=44" example:"public key for the metadataV2 encoded to base64url"`
	Timestamp     int64  `json:"timestamp" validate:"required"`
}

type metadataMultipleV2KeyObject struct {
	MetadataV2Keys []string `json:"metadataV2Keys" validate:"gt=0,required,dive,base64url" example:"public keys for the metadataV2 encoded to base64url"`
	Timestamp      int64    `json:"timestamp" validate:"required"`
}

type metadataV2KeyReq struct {
	verification
	requestBody
	metadataV2KeyObject metadataV2KeyObject
}

type metadataMultipleV2KeyReq struct {
	verification
	requestBody
	metadataMultipleV2KeyObject metadataMultipleV2KeyObject
}

type metadataV2PublicKeyReq struct {
	requestBody
	metadataV2KeyObject metadataV2KeyObject
}

type getMetadataV2Res struct {
	MetadataV2     string    `json:"metadataV2" validate:"required,base64url,omitempty" example:"your account metadataV2"`
	ExpirationDate time.Time `json:"expirationDate" validate:"required"`
}

type createMetadataV2Res struct {
	ExpirationDate time.Time `json:"expirationDate" validate:"required"`
}

var metadataV2DeletedRes = StatusRes{
	Status: "metadataV2 successfully deleted",
}

func (v *updateMetadataV2Req) getObjectRef() interface{} {
	return &v.updateMetadataV2Object
}

func (v *updateMetadataMultipleV2Req) getObjectRef() interface{} {
	return &v.updateMetadataMultipleV2Object
}

func (v *metadataV2KeyReq) getObjectRef() interface{} {
	return &v.metadataV2KeyObject
}

func (v *metadataMultipleV2KeyReq) getObjectRef() interface{} {
	return &v.metadataMultipleV2KeyObject
}

// GetMetadataV2Handler godoc
// @Summary Retrieve account metadataV2
// @Accept  json
// @Produce  json
// @Param metadataV2KeyReq body routes.metadataV2KeyReq true "object for endpoints that only need metadataV2Key and timestamp"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"metadataV2Key": "public key for the metadataV2 encoded to base64",
// @description 	"timestamp": 1557346389
// @description }
// @Success 200 {object} routes.getMetadataV2Res
// @Failure 404 {string} string "no value found for that key, or account not found"
// @Failure 403 {string} string "subscription expired, or the invoice resonse"
// @Failure 400 {string} string "bad request, unable to parse b64: (with the error)"
// @Failure 400 {string} string "bad request, incorrect key length"
// @Router /api/v2/metadata/get [post]
/*GetMetadataV2Handler is a handler for getting the file metadataV2*/
func GetMetadataV2Handler() gin.HandlerFunc {
	return ginHandlerFunc(getMetadataV2)
}

// GetMetadataV2PublicHandler godoc
// @Summary Retrieve account metadataV2
// @Accept  json
// @Produce  json
// @Param metadataV2PublicKeyReq body routes.metadataV2PublicKeyReq true "object for endpoints that only need metadataV2Key and timestamp"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"metadataV2Key": "public key for the metadataV2 encoded to base64",
// @description 	"timestamp": 1557346389
// @description }
// @Success 200 {object} routes.getMetadataV2Res
// @Failure 404 {string} string "no value found for that key, or account not found"
// @Failure 403 {string} string "subscription expired, or the invoice resonse"
// @Failure 400 {string} string "bad request, unable to parse b64: (with the error)"
// @Failure 400 {string} string "bad request, incorrect key length"
// @Router /api/v2/metadata/get-public [post]
/*GetMetadataV2PublicHandler is a handler for getting the public file metadataV2*/
func GetMetadataV2PublicHandler() gin.HandlerFunc {
	return ginHandlerFunc(getMetadataV2Public)
}

// UpdateMetadataV2Handler godoc
// @Summary Update metadataV2
// @Accept  json
// @Produce  json
// @Param updateMetadataV2Req body routes.updateMetadataV2Req true "update metadataV2 object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"metadataV2Key": "public key for the metadataV2 encoded to base64",
// @description 	"metadataV2Vertex": "the vertex to add to your account metadataV2 encoded to base64",
// @description 	"metadataV2Edges": "the edges to add to your account metadataV2 encoded to base64",
// @description 	"metadataV2Sig": "a signature encoded to base64 confirming the metadata change, the publickey will be a key for the metadataV2",
// @description 	"timestamp": 1557346389
// @description }
// @Success 200 {object} routes.updateMetadataV2Res
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 400 {string} string "bad request, unable to parse vertex: (with the error)"
// @Failure 400 {string} string "bad request, unable to parse edge: (with the error)"
// @Failure 400 {string} string "bad request, unable to add edge to dag: (with the error)"
// @Failure 400 {string} string "bad request, can't verify signature: (with the error)"
// @Failure 404 {string} string "no value found for that key, or account not found"
// @Failure 403 {string} string "subscription expired, or the invoice response"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v2/metadata/add [post]
/*UpdateMetadataV2Handler is a handler for updating the file metadataV2*/
func UpdateMetadataV2Handler() gin.HandlerFunc {
	return ginHandlerFunc(updateMetadataV2)
}

// UpdateMetadataMultipleV2Handler godoc
// @Summary Update multiple metadataV2
// @Accept json
// @Produce json
// @Param updateMetadataMultipleV2Req body routes.updateMetadataMultipleV2Req true "update metadataV2 objects"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description   "metadatas": [{
// @description 	  "metadataV2Key": "public key for the metadataV2 encoded to base64",
// @description 	  "metadataV2Vertex": "the vertex to add to your account metadataV2 encoded to base64",
// @description 	  "metadataV2Edges": "the edges to add to your account metadataV2 encoded to base64",
// @description 	  "metadataV2Sig": "a signature encoded to base64 confirming the metadata change, the publickey will be a key for the metadataV2",
// @description   },
// @description		{ ... }]
// @description 	"timestamp": 1557346389
// @description }
// @Success 200 {object} routes.updateMetadataMultipleV2Res
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 400 {string} string "bad request, unable to parse vertex: (with the error)"
// @Failure 400 {string} string "bad request, unable to parse edge: (with the error)"
// @Failure 400 {string} string "bad request, unable to add edge to dag: (with the error)"
// @Failure 400 {string} string "bad request, can't verify signature: (with the error)"
// @Failure 404 {string} string "no value found for that key, or account not found"
// @Failure 403 {string} string "subscription expired, or the invoice response"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v2/metadata/add-multiple [post]
/*UpdateMetadataMultipleV2Handler is a handler for updating multiple file metadataV2*/
func UpdateMetadataMultipleV2Handler() gin.HandlerFunc {
	return ginHandlerFunc(updateMetadataMultipleV2)
}

// DeleteMetadataV2Handler godoc
// @Summary delete a metadataV2
// @Accept json
// @Produce json
// @Param metadataV2KeyReq body routes.metadataV2KeyReq true "object for endpoints that only need metadataV2Key and timestamp"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"metadataV2Key": "public key for the metadataV2 encoded to base64",
// @description 	"timestamp": 1557346389
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 404 {string} string "account not found"
// @Failure 403 {string} string "subscription expired, or the invoice resonse"
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 400 {string} string "bad request, unable to parse b64: (with the error)"
// @Failure 400 {string} string "bad request, incorrect key length"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v2/metadata/delete [post]
/*DeleteMetadataV2Handler is a handler for deleting a metadataV2*/
func DeleteMetadataV2Handler() gin.HandlerFunc {
	return ginHandlerFunc(deleteMetadataV2)
}

// DeleteMetadataMultipleV2Handler godoc
// @Summary delete multiple metadataV2, if a key is not found, it won't be treated as an error
// @Accept json
// @Produce json
// @Param metadataMultipleV2KeyReq body routes.metadataMultipleV2KeyReq true "object for endpoint that only needs an array of metadataV2Keys and timestamp"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"metadataV2Keys": ["public key for the metadataV2 encoded to base64", "another public key for the metadataV2 encoded to base64", "..."],
// @description 	"timestamp": 1557346389
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 404 {string} string "account not found"
// @Failure 403 {string} string "subscription expired, or the invoice resonse"
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 400 {string} string "bad request, unable to parse b64: (with the error)"
// @Failure 400 {string} string "bad request, incorrect key length"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v2/metadata/delete-multiple [post]
/*DeleteMetadataMultipleV2Handler is a handler for deleting a metadataV2*/
func DeleteMetadataMultipleV2Handler() gin.HandlerFunc {
	return ginHandlerFunc(deleteMetadataMultipleV2)
}

func getMetadataV2(c *gin.Context) error {
	request := metadataV2KeyReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	if paid, _ := verifyIfPaid(account); !paid {
		cost, _ := account.Cost()
		return AccountNotPaidResponse(c, accountCreateRes{
			Invoice: models.Invoice{
				Cost:       cost,
				EthAddress: account.EthAddress,
			},
			ExpirationDate: account.ExpirationDate(),
		})
	}

	requestBodyParsed := metadataV2KeyObject{}

	if err := verifyAndParseStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c); err != nil {
		return err
	}

	metadataV2KeyBin, err := base64.URLEncoding.DecodeString(requestBodyParsed.MetadataV2Key)
	if err != nil {
		err = fmt.Errorf("bad request, unable to parse b64: %v", err)
		return BadRequestResponse(c, err)
	}

	if cap(metadataV2KeyBin) != 33 {
		return BadRequestResponse(c, errors.New(metadataIncorrectKeyLength))
	}

	permissionHashKey := getPermissionHashV2KeyForBadger(string(metadataV2KeyBin))
	permissionHashInBadger, _, err := utils.GetValueFromKV(permissionHashKey)

	if err != nil {
		return NotFoundResponse(c, err)
	}

	publicKeyBin, err := hex.DecodeString(request.PublicKey)
	if err != nil {
		err = fmt.Errorf("bad request, unable to parse hex: %v", err)
		return BadRequestResponse(c, err)
	}

	if err := verifyPermissionsV2(publicKeyBin, metadataV2KeyBin, permissionHashInBadger, c); err != nil {
		return err
	}

	metadataV2, expirationTime, err := utils.GetValueFromKV(string(metadataV2KeyBin))

	if err != nil {
		return NotFoundResponse(c, err)
	}

	return OkResponse(c, getMetadataV2Res{
		MetadataV2:     metadataV2,
		ExpirationDate: expirationTime,
	})
}

func getMetadataV2Public(c *gin.Context) error {
	request := metadataV2PublicKeyReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	requestBodyParsed := metadataV2KeyObject{}

	if err := parseStringRequest(request.RequestBody, &requestBodyParsed, c); err != nil {
		return err
	}

	metadataV2KeyBin, err := base64.URLEncoding.DecodeString(requestBodyParsed.MetadataV2Key)
	if err != nil {
		err = fmt.Errorf("bad request, unable to parse b64: %v", err)
		return BadRequestResponse(c, err)
	}

	if cap(metadataV2KeyBin) != 33 {
		return BadRequestResponse(c, errors.New(metadataIncorrectKeyLength))
	}

	isPublicKey := getIsPublicV2KeyForBadger(string(metadataV2KeyBin))
	isPublicInBadger, _, err := utils.GetValueFromKV(isPublicKey)

	if err != nil {
		return NotFoundResponse(c, err)
	}

	if isPublicInBadger != "true" {
		return NotFoundResponse(c, errors.New("key not found"))
	}

	metadataV2, expirationTime, err := utils.GetValueFromKV(string(metadataV2KeyBin))

	if err != nil {
		return NotFoundResponse(c, err)
	}

	return OkResponse(c, getMetadataV2Res{
		MetadataV2:     metadataV2,
		ExpirationDate: expirationTime,
	})
}

func updateMetadataV2(c *gin.Context) error {
	request := updateMetadataV2Req{}

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

	requestBodyParsed := updateMetadataV2Object{}

	if err := verifyAndParseStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c); err != nil {
		return err
	}

	metadataV2KeyBin, err := base64.URLEncoding.DecodeString(requestBodyParsed.MetadataV2Key)
	if err != nil {
		err = fmt.Errorf("bad request, unable to parse b64: %v", err)
		return BadRequestResponse(c, err)
	}

	if cap(metadataV2KeyBin) != 33 {
		return BadRequestResponse(c, errors.New(metadataIncorrectKeyLength))
	}

	publicKeyBin, err := hex.DecodeString(request.PublicKey)
	if err != nil {
		err = fmt.Errorf("bad request, unable to parse hex: %v", err)
		return BadRequestResponse(c, err)
	}

	// Setting ttls on metadata to 2 months post account expiration date so the metadatas won't
	// be deleted too soon
	ttl := time.Until(account.ExpirationDate().Add(MetadataExpirationOffset))

	oldMetadataV2, _, err := utils.GetValueFromKV(string(metadataV2KeyBin))

	if err != nil {
		if err = account.IncrementMetadataCount(); err != nil {
			return ForbiddenResponse(c, err)
		}

		permissionHash := getPermissionHashV2(publicKeyBin, metadataV2KeyBin)
		permissionHashKey := getPermissionHashV2KeyForBadger(string(metadataV2KeyBin))

		isPublicKey := getIsPublicV2KeyForBadger(string(metadataV2KeyBin))

		d := dag.NewDAG()

		if err = utils.BatchSet(&utils.KVPairs{
			string(metadataV2KeyBin): base64.URLEncoding.EncodeToString(d.Binary()),
			permissionHashKey:        permissionHash,
			isPublicKey:              strconv.FormatBool(requestBodyParsed.IsPublic),
		}, ttl); err != nil {
			account.DecrementMetadataCount()
			return InternalErrorResponse(c, err)
		}

		oldMetadataV2 = base64.URLEncoding.EncodeToString(d.Binary())
	}

	permissionHashKey := getPermissionHashV2KeyForBadger(string(metadataV2KeyBin))
	permissionHashInBadger, _, err := utils.GetValueFromKV(permissionHashKey)

	if err != nil {
		return NotFoundResponse(c, err)
	}

	if err := verifyPermissionsV2(publicKeyBin, metadataV2KeyBin,
		permissionHashInBadger, c); err != nil {
		return err
	}

	isPublicKey := getIsPublicV2KeyForBadger(string(metadataV2KeyBin))
	isPublicInBadger, _, err := utils.GetValueFromKV(isPublicKey)

	if err != nil {
		return NotFoundResponse(c, err)
	}

	if isPublicInBadger != strconv.FormatBool(requestBodyParsed.IsPublic) {
		return BadRequestResponse(c, errors.New("bad request, isPublic does not match"))
	}

	dBin, err := base64.URLEncoding.DecodeString(oldMetadataV2)

	if err != nil {
		err = fmt.Errorf("bad request, unable to parse b64: %v", err)
		return BadRequestResponse(c, err)
	}

	dagFromBinary, err := dag.NewDAGFromBinary(dBin)

	if err != nil {
		return InternalErrorResponse(c, err)
	}

	vBin, err := base64.URLEncoding.DecodeString(requestBodyParsed.MetadataV2Vertex)

	if err != nil {
		err = fmt.Errorf("bad request, unable to parse b64: %v", err)
		return BadRequestResponse(c, err)
	}

	vert, err := dag.NewDAGVertexFromBinary(vBin)

	if err != nil {
		err = fmt.Errorf("bad request, unable to parse vertex: %v", err)
		return BadRequestResponse(c, err)
	}

	dagFromBinary.Add(*vert)

	for _, eB64 := range requestBodyParsed.MetadataV2Edges {
		eBin, err := base64.URLEncoding.DecodeString(eB64)

		if err != nil {
			err = fmt.Errorf("bad request, unable to parse b64: %v", err)
			return BadRequestResponse(c, err)
		}

		edge, err := dag.NewDAGEdgeFromBinary(eBin)

		if err != nil {
			err = fmt.Errorf("bad request, unable to parse edge: %v", err)
			return BadRequestResponse(c, err)
		}

		err = dagFromBinary.AddEdge(*edge)

		if err != nil {
			err = fmt.Errorf("bad request, unable to add edge to dag: %v", err)
			return BadRequestResponse(c, err)
		}
	}

	vDigest, err := dagFromBinary.Digest(vert.ID, dag.DigestHashSHA256)

	if err != nil {
		return InternalErrorResponse(c, err)
	}

	metadataV2SigBin, err := base64.URLEncoding.DecodeString(requestBodyParsed.MetadataV2Sig)

	if err != nil {
		err = fmt.Errorf("bad request, unable to parse b64: %v", err)
		return BadRequestResponse(c, err)
	}

	if !secp256k1.VerifySignature(metadataV2KeyBin, vDigest, []byte(metadataV2SigBin)) {
		err = fmt.Errorf("bad request, can't verify signature: %v", err)
		return BadRequestResponse(c, err)
	}

	if account.ExpirationDate().Before(time.Now()) {
		return ForbiddenResponse(c, errors.New("subscription expired"))
	}

	newMetadataV2 := base64.URLEncoding.EncodeToString(dagFromBinary.Binary())

	if err := account.UpdateMetadataSizeInBytes(int64(len(oldMetadataV2)), int64(len(newMetadataV2))); err != nil {
		return ForbiddenResponse(c, err)
	}

	if err := utils.BatchSet(&utils.KVPairs{
		string(metadataV2KeyBin): newMetadataV2,
		permissionHashKey:        permissionHashInBadger,
	}, ttl); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, updateMetadataV2Res{
		updateMetadataV2ResBase: updateMetadataV2ResBase{
			MetadataV2Key: request.updateMetadataV2Object.MetadataV2Key,
			MetadataV2:    newMetadataV2,
		},
		ExpirationDate: account.ExpirationDate().Add(MetadataExpirationOffset),
	})
}

func updateMetadataMultipleV2(c *gin.Context) error {
	return nil
}

func deleteMetadataV2(c *gin.Context) error {
	request := metadataV2KeyReq{}

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

	requestBodyParsed := metadataV2KeyObject{}

	if err := verifyAndParseStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c); err != nil {
		return err
	}

	metadataV2KeyBin, err := base64.URLEncoding.DecodeString(requestBodyParsed.MetadataV2Key)
	if err != nil {
		err = fmt.Errorf("bad request, unable to parse b64: %v", err)
		return BadRequestResponse(c, err)
	}

	if cap(metadataV2KeyBin) != 33 {
		return BadRequestResponse(c, errors.New(metadataIncorrectKeyLength))
	}

	publicKeyBin, err := hex.DecodeString(request.PublicKey)
	if err != nil {
		err = fmt.Errorf("bad request, unable to parse hex: %v", err)
		return BadRequestResponse(c, err)
	}

	permissionHashKey := getPermissionHashV2KeyForBadger(string(metadataV2KeyBin))
	permissionHashInBadger, _, err := utils.GetValueFromKV(permissionHashKey)

	if err != nil {
		return NotFoundResponse(c, err)
	}

	if err := verifyPermissionsV2(publicKeyBin, metadataV2KeyBin, permissionHashInBadger, c); err != nil {
		return err
	}

	oldMetadataV2, _, err := utils.GetValueFromKV(string(metadataV2KeyBin))
	if err != nil {
		return NotFoundResponse(c, err)
	}

	if err := account.RemoveMetadata(int64(len(oldMetadataV2))); err != nil {
		return InternalErrorResponse(c, err)
	}

	if err = utils.BatchDelete(&utils.KVKeys{
		string(metadataV2KeyBin),
		permissionHashKey,
	}); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, metadataV2DeletedRes)
}

func deleteMetadataMultipleV2(c *gin.Context) error {
	request := metadataMultipleV2KeyReq{}

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

	requestBodyParsed := metadataMultipleV2KeyObject{}

	if err := verifyAndParseStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c); err != nil {
		return err
	}

	publicKeyBin, err := hex.DecodeString(request.PublicKey)
	if err != nil {
		err = fmt.Errorf("bad request, unable to parse hex: %v", err)
		return BadRequestResponse(c, err)
	}

	metadataV2KeyBins := make(map[string][]byte, len(requestBodyParsed.MetadataV2Keys))

	var metadataKvKeys utils.KVKeys
	var permissionHashKvKeys utils.KVKeys

	for _, metadataV2Key := range requestBodyParsed.MetadataV2Keys {
		metadataV2KeyBin, err := base64.URLEncoding.DecodeString(metadataV2Key)
		if err != nil {
			err = fmt.Errorf("bad request, unable to parse b64: %v", err)
			return BadRequestResponse(c, err)
		}

		if cap(metadataV2KeyBin) != 33 {
			return BadRequestResponse(c, errors.New(metadataIncorrectKeyLength))
		}

		metadataV2KeyBins[metadataV2Key] = metadataV2KeyBin
		metadataKvKeys = append(metadataKvKeys, string(metadataV2KeyBin))

		permissionHashKey := getPermissionHashV2KeyForBadger(string(metadataV2KeyBin))
		permissionHashKvKeys = append(permissionHashKvKeys, permissionHashKey)
	}

	permissionHashKvPairs, err := utils.BatchGet(&permissionHashKvKeys)
	if err != nil {
		return NotFoundResponse(c, err)
	}
	oldMetadatasV2, err := utils.BatchGet(&metadataKvKeys)
	if err != nil {
		return NotFoundResponse(c, err)
	}

	lenOldMetadatasV2 := 0
	countOldMetadatasV2 := 0
	var deleteKvKeys utils.KVKeys
	for _, metadataV2Key := range requestBodyParsed.MetadataV2Keys {
		metadataV2KeyBinsString := string(metadataV2KeyBins[metadataV2Key])
		permissionHashInBadgerKey := getPermissionHashV2KeyForBadger(metadataV2KeyBinsString)
		if permissionHashInBadger, ok := (*permissionHashKvPairs)[permissionHashInBadgerKey]; ok {
			if err := verifyPermissionsV2(publicKeyBin, metadataV2KeyBins[metadataV2Key], permissionHashInBadger, c); err != nil {
				return err
			}
			deleteKvKeys = append(deleteKvKeys, permissionHashInBadgerKey)
		}

		if oldMetadataV2, ok := (*oldMetadatasV2)[metadataV2KeyBinsString]; ok {
			lenOldMetadatasV2 += len(oldMetadataV2)
			countOldMetadatasV2++
			deleteKvKeys = append(deleteKvKeys, metadataV2KeyBinsString)
		}
	}

	if err := account.RemoveMetadataMultiple(int64(lenOldMetadatasV2), countOldMetadatasV2); err != nil {
		return InternalErrorResponse(c, err)
	}

	if err = utils.BatchDelete(&deleteKvKeys); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, metadataV2DeletedRes)
}
