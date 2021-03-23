package routes

import (
	"crypto/sha256"
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

// must be sorted alphabetically for JSON marshaling/stringifying
type updateMetadataV2Object struct {
	MetadataV2Edges  []string `json:"metadataV2Edges" binding:"required,dive,required,base64,len=12" example:"the edges to add to your account metadataV2 encoded to base64"`
	MetadataV2Key    string   `json:"metadataV2Key" binding:"required,base64,len=44" example:"public key for the metadataV2 encoded to base64"`
	MetadataV2Sig    string   `json:"metadataV2Sig" binding:"required,base64,len=88" example:"a signature encoded to base64 confirming the metadata change, the publickey will be a key for the metadataV2"`
	MetadataV2Vertex string   `json:"metadataV2Vertex" binding:"required,base64" example:"the vertex to add to your account metadataV2 encoded to base64"`
	IsPublic         bool     `json:"isPublic" binding:"required"`
	Timestamp        int64    `json:"timestamp" binding:"required"`
}

type updateMetadataV2Req struct {
	verification
	requestBody
	updateMetadataV2Object updateMetadataV2Object
}

type updateMetadataV2Res struct {
	MetadataV2Key  string    `json:"metadataV2Key" binding:"required,base64,len=44" example:"public key for the metadataV2 encoded to base64"`
	MetadataV2     string    `json:"metadataV2" binding:"required,base64" example:"your (updated) account metadataV2"`
	ExpirationDate time.Time `json:"expirationDate" binding:"required,gte"`
}

type metadataV2KeyObject struct {
	MetadataV2Key string `json:"metadataV2Key" binding:"required,base64,len=44" example:"public key for the metadataV2 encoded to base64"`
	Timestamp     int64  `json:"timestamp" binding:"required"`
}

type metadataV2KeyReq struct {
	verification
	requestBody
	metadataV2KeyObject metadataV2KeyObject
}

type metadataV2PublicKeyReq struct {
	requestBody
	metadataV2KeyObject metadataV2KeyObject
}

type getMetadataV2Res struct {
	MetadataV2     string    `json:"metadataV2" binding:"exists,base64,omitempty" example:"your account metadataV2"`
	ExpirationDate time.Time `json:"expirationDate" binding:"required"`
}

type createMetadataV2Res struct {
	ExpirationDate time.Time `json:"expirationDate" binding:"required"`
}

var metadataV2DeletedRes = StatusRes{
	Status: "metadataV2 successfully deleted",
}

func (v *updateMetadataV2Req) getObjectRef() interface{} {
	return &v.updateMetadataV2Object
}

func (v *metadataV2KeyReq) getObjectRef() interface{} {
	return &v.metadataV2KeyObject
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
	return ginHandlerFunc(getMetadataV2)
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

// DeleteMetadataV2Handler godoc
// @Summary delete a metadataV2
// @Accept  json
// @Produce  json
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

func getMetadataV2(c *gin.Context) error {
	request := metadataV2KeyReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	if paid := verifyIfPaid(account); !paid {
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

	metadataV2KeyBin, err := base64.StdEncoding.DecodeString(requestBodyParsed.MetadataV2Key)
	if err != nil {
		err = fmt.Errorf("bad request, unable to parse b64: %v", err)
		return BadRequestResponse(c, err)
	}

	if len(metadataV2KeyBin) != 33 {
		return BadRequestResponse(c, errors.New("bad request, incorrect key length"))
	}

	permissionHashKey := getPermissionHashV2KeyForBadger(requestBodyParsed.MetadataV2Key)
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

	metadataV2, expirationTime, err := utils.GetValueFromKV(request.metadataV2KeyObject.MetadataV2Key)

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

	metadataV2KeyBin, err := base64.StdEncoding.DecodeString(requestBodyParsed.MetadataV2Key)
	if err != nil {
		err = fmt.Errorf("bad request, unable to parse b64: %v", err)
		return BadRequestResponse(c, err)
	}

	if len(metadataV2KeyBin) != 33 {
		return BadRequestResponse(c, errors.New("bad request, incorrect key length"))
	}

	isPublicKey := getIsPublicV2KeyForBadger(requestBodyParsed.MetadataV2Key)
	isPublicInBadger, _, err := utils.GetValueFromKV(isPublicKey)

	if err != nil {
		return NotFoundResponse(c, err)
	}

	if isPublicInBadger != "true" {
		return NotFoundResponse(c, errors.New("Key not found"))
	}

	metadataV2, expirationTime, err := utils.GetValueFromKV(request.metadataV2KeyObject.MetadataV2Key)

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

	metadataV2KeyBin, err := base64.StdEncoding.DecodeString(requestBodyParsed.MetadataV2Key)
	if err != nil {
		err = fmt.Errorf("bad request, unable to parse b64: %v", err)
		return BadRequestResponse(c, err)
	}

	if len(metadataV2KeyBin) != 33 {
		return BadRequestResponse(c, errors.New("bad request, incorrect key length"))
	}

	oldMetadataV2, _, err := utils.GetValueFromKV(requestBodyParsed.MetadataV2Key)

	publicKeyBin, err := hex.DecodeString(request.PublicKey)
	if err != nil {
		err = fmt.Errorf("bad request, unable to parse hex: %v", err)
		return BadRequestResponse(c, err)
	}

	if err != nil {
		if err = account.IncrementMetadataCount(); err != nil {
			return ForbiddenResponse(c, err)
		}

		ttl := time.Until(account.ExpirationDate())

		permissionHash := getPermissionHashV2(publicKeyBin, metadataV2KeyBin, c)
		permissionHashKey := getPermissionHashV2KeyForBadger(requestBodyParsed.MetadataV2Key)

		isPublicKey := getIsPublicV2KeyForBadger(requestBodyParsed.MetadataV2Key)

		d := dag.NewDAG()

		if err = utils.BatchSet(&utils.KVPairs{
			requestBodyParsed.MetadataV2Key: base64.StdEncoding.EncodeToString(d.Binary()),
			permissionHashKey:               permissionHash,
			isPublicKey:                     strconv.FormatBool(requestBodyParsed.IsPublic),
		}, ttl); err != nil {
			account.DecrementMetadataCount()
			return InternalErrorResponse(c, err)
		}

		oldMetadataV2 = base64.StdEncoding.EncodeToString(d.Binary())
	}

	permissionHashKey := getPermissionHashV2KeyForBadger(requestBodyParsed.MetadataV2Key)
	permissionHashInBadger, _, err := utils.GetValueFromKV(permissionHashKey)

	if err != nil {
		return NotFoundResponse(c, err)
	}

	if err := verifyPermissionsV2(publicKeyBin, metadataV2KeyBin,
		permissionHashInBadger, c); err != nil {
		return err
	}

	isPublicKey := getIsPublicV2KeyForBadger(requestBodyParsed.MetadataV2Key)
	isPublicInBadger, _, err := utils.GetValueFromKV(isPublicKey)

	if err != nil {
		return NotFoundResponse(c, err)
	}

	if isPublicInBadger != strconv.FormatBool(requestBodyParsed.IsPublic) {
		return BadRequestResponse(c, errors.New("bad request, isPublic does not match"))
	}

	dBin, err := base64.StdEncoding.DecodeString(oldMetadataV2)

	if err != nil {
		err = fmt.Errorf("bad request, unable to parse b64: %v", err)
		return BadRequestResponse(c, err)
	}

	d, err := dag.NewDAGFromBinary(dBin)

	if err != nil {
		return InternalErrorResponse(c, err)
	}

	vBin, err := base64.StdEncoding.DecodeString(requestBodyParsed.MetadataV2Vertex)

	if err != nil {
		err = fmt.Errorf("bad request, unable to parse b64: %v", err)
		return BadRequestResponse(c, err)
	}

	vert, err := dag.NewDAGVertexFromBinary(vBin)

	if err != nil {
		err = fmt.Errorf("bad request, unable to parse vertex: %v", err)
		return BadRequestResponse(c, err)
	}

	d.Add(*vert)

	for _, eB64 := range requestBodyParsed.MetadataV2Edges {
		eBin, err := base64.StdEncoding.DecodeString(eB64)

		if err != nil {
			err = fmt.Errorf("bad request, unable to parse b64: %v", err)
			return BadRequestResponse(c, err)
		}

		edge, err := dag.NewDAGEdgeFromBinary(eBin)

		if err != nil {
			err = fmt.Errorf("bad request, unable to parse edge: %v", err)
			return BadRequestResponse(c, err)
		}

		err = d.AddEdge(*edge)

		if err != nil {
			err = fmt.Errorf("bad request, unable to add edge to dag: %v", err)
			return BadRequestResponse(c, err)
		}
	}

	vDigest, err := d.Digest(vert.ID, func(b []byte) []byte { s := sha256.Sum256(b); return s[:] })

	if err != nil {
		return InternalErrorResponse(c, err)
	}

	metadataV2SigBin, err := base64.StdEncoding.DecodeString(requestBodyParsed.MetadataV2Sig)

	if err != nil {
		err = fmt.Errorf("bad request, unable to parse b64: %v", err)
		return BadRequestResponse(c, err)
	}

	if secp256k1.VerifySignature(metadataV2KeyBin, vDigest, []byte(metadataV2SigBin)) == false {
		err = fmt.Errorf("bad request, can't verify signature: %v", err)
		return BadRequestResponse(c, err)
	}

	if account.ExpirationDate().Before(time.Now()) {
		return ForbiddenResponse(c, errors.New("subscription expired"))
	}

	newMetadataV2 := base64.StdEncoding.EncodeToString(d.Binary())

	if err := account.UpdateMetadataSizeInBytes(int64(len(oldMetadataV2)), int64(len(newMetadataV2))); err != nil {
		return ForbiddenResponse(c, err)
	}

	ttl := time.Until(account.ExpirationDate())

	if err := utils.BatchSet(&utils.KVPairs{
		requestBodyParsed.MetadataV2Key: newMetadataV2,
		permissionHashKey:               permissionHashInBadger,
	}, ttl); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, updateMetadataV2Res{
		MetadataV2Key:  request.updateMetadataV2Object.MetadataV2Key,
		MetadataV2:     newMetadataV2,
		ExpirationDate: account.ExpirationDate(),
	})
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

	metadataV2KeyBin, err := base64.StdEncoding.DecodeString(requestBodyParsed.MetadataV2Key)
	if err != nil {
		err = fmt.Errorf("bad request, unable to parse b64: %v", err)
		return BadRequestResponse(c, err)
	}

	if len(metadataV2KeyBin) != 33 {
		return BadRequestResponse(c, errors.New("bad request, incorrect key length"))
	}

	publicKeyBin, err := hex.DecodeString(request.PublicKey)
	if err != nil {
		err = fmt.Errorf("bad request, unable to parse hex: %v", err)
		return BadRequestResponse(c, err)
	}

	permissionHashKey := getPermissionHashV2KeyForBadger(requestBodyParsed.MetadataV2Key)
	permissionHashInBadger, _, err := utils.GetValueFromKV(permissionHashKey)

	if err != nil {
		return NotFoundResponse(c, err)
	}

	if err := verifyPermissionsV2(publicKeyBin, metadataV2KeyBin, permissionHashInBadger, c); err != nil {
		return err
	}

	oldMetadataV2, _, err := utils.GetValueFromKV(requestBodyParsed.MetadataV2Key)

	if err := account.RemoveMetadata(int64(len(oldMetadataV2))); err != nil {
		return InternalErrorResponse(c, err)
	}

	if err = utils.BatchDelete(&utils.KVKeys{
		requestBodyParsed.MetadataV2Key,
		permissionHashKey,
	}); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, metadataV2DeletedRes)
}
