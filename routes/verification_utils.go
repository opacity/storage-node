package routes

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

const (
	signatureDidNotMatchResponse = "signature did not match"
	errVerifying                 = "error verifying signature"
	marshalError                 = "bad request, unable to marshal request body: "
	notAuthorizedResponse        = "you are not authorized to access or modify this resource"
	postFormTag                  = "form"
	postFormFileTag              = "formFile"
	bindingTag                   = "binding"
)

type verificationInterface interface {
	getVerification() verification
	getAccount(c *gin.Context) (models.Account, error)
}

type parsableObjectInterface interface {
	// Return the reference of object
	getObjectRef() interface{}
	getObjectAsString() string
}

type verification struct {
	// signature without 0x prefix is broken into
	// R: sig[0:63]
	// S: sig[64:127]
	Signature string `json:"signature" form:"signature" binding:"required,len=128" minLength:"128" maxLength:"128" example:"a 128 character string created when you signed the request with your private key or account handle"`
	PublicKey string `json:"publicKey" form:"publicKey" binding:"required,len=66" minLength:"66" maxLength:"66" example:"a 66-character public key"`
}

func (v verification) getVerification() verification {
	return v
}

func (v verification) getAccountId(c *gin.Context) (string, error) {
	accountId, err := utils.HashString(v.PublicKey)
	if err != nil {
		return "", InternalErrorResponse(c, err)
	}
	return accountId, err
}

func (v verification) getAccount(c *gin.Context) (models.Account, error) {
	accountID, err := v.getAccountId(c)
	if err != nil {
		return models.Account{}, err
	}

	// validate user
	account, err := models.GetAccountById(accountID)
	if err != nil || len(account.AccountID) == 0 {
		return account, AccountNotFoundResponse(c, accountID)
	}

	return account, err
}

type requestBody struct {
	RequestBody string `json:"requestBody" form:"requestBody" binding:"required" example:"look at description for example"`
}

func (v requestBody) getObjectAsString() string {
	return v.RequestBody
}

func verifyAndParseBodyRequest(dest interface{}, c *gin.Context) error {
	if err := utils.ParseRequestBody(c.Request, dest); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		return BadRequestResponse(c, err)
	}

	if i, ok := dest.(verificationInterface); ok {
		if ii, ok := dest.(parsableObjectInterface); ok {
			return verifyAndParseStringRequest(ii.getObjectAsString(), ii.getObjectRef(), i.getVerification(), c)
		}
	}
	return nil
}

func verifyAndParseFormRequest(dest interface{}, c *gin.Context) error {
	defer c.Request.Body.Close()
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxRequestSize)
	err := c.Request.ParseMultipartForm(MaxRequestSize)
	if err != nil {
		return BadRequestResponse(c, err)
	}

	t := reflect.ValueOf(dest).Elem().Type()
	s := reflect.ValueOf(dest).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i) // Get the field, returns https://golang.org/pkg/reflect/#StructField
		if field.Anonymous {
			// Only go 1 level down
			for j := 0; j < field.Type.NumField(); j++ {
				nestField := field.Type.Field(j)
				strV, err := getValueFromPostForm(nestField, c)
				if err != nil {
					return err
				}
				if strV == "" {
					continue
				}
				if !s.Field(i).Field(j).CanSet() {
					return InternalErrorResponse(c, fmt.Errorf("Field is not settable, It should be upper case but has this: %v", nestField))
				}
				s.Field(i).Field(j).SetString(strV)
			}
		}

		strV, err := getValueFromPostForm(field, c)
		if err != nil {
			return err
		}
		if strV == "" {
			continue
		}
		if !s.Field(i).CanSet() {
			return InternalErrorResponse(c, fmt.Errorf("Field is not settable, It should be upper case but has this: %v", field))
		}
		s.Field(i).SetString(strV)
	}

	if i, ok := dest.(verificationInterface); ok {
		if ii, ok := dest.(parsableObjectInterface); ok {
			return verifyAndParseStringRequest(ii.getObjectAsString(), ii.getObjectRef(), i.getVerification(), c)
		}
	}
	return nil
}

func getValueFromPostForm(field reflect.StructField, c *gin.Context) (string, error) {
	strV := ""
	formTag := field.Tag.Get(postFormTag)
	fileTag := field.Tag.Get(postFormFileTag)
	binding := field.Tag.Get(bindingTag)
	required := strings.Contains(binding, "required")

	if formTag == "" && fileTag == "" {
		return "", nil
	}
	if formTag != "" {
		strV = c.Request.FormValue(formTag)
	}

	if fileTag != "" {
		v, err := readFileFromForm(fileTag)
		if err != nil && required {
			return "", BadRequestResponse(c, err)
		}
		// otherwise, just ignore the error
		strV = v
	}
	return strV, nil
}

func readFileFromForm(fileTag string) (string, error) {
	multiFile, _, err := c.Request.FormFile(fileTag)
	defer func() {
		if multiFile != nil {
			multiFile.Close()
		}
	}()

	if err != nil {
		return "", fmt.Errorf("Unable to get file %v from POST form", fileTag)
	}
	var fileBytes bytes.Buffer
	if _, err := io.Copy(&fileBytes, multiFile); err != nil {
		return "", err
	}
	return fileBytes.String(), nil
}

func verifyAndParseStringRequest(reqAsString string, dest interface{}, verificationData verification, c *gin.Context) error {
	hash := utils.Hash([]byte(reqAsString))

	if err := verifyRequest(hash, verificationData, c); err != nil {
		return err
	}

	if err := utils.ParseStringifiedRequest(reqAsString, dest); err != nil {
		return BadRequestResponse(c, fmt.Errorf("bad request, unable to parse request body: %v", err))
	}

	return nil
}

func verifyParsedRequest(reqBody interface{}, verificationData verification, c *gin.Context) error {
	hash, err := hashRequestBody(reqBody, c)
	if err != nil {
		return err
	}

	if err := verifyRequest(hash, verificationData, c); err != nil {
		return err
	}

	return nil
}

func verifyRequest(hash []byte, verificationData verification, c *gin.Context) error {
	verified, err := utils.VerifyFromStrings(verificationData.PublicKey, hex.EncodeToString(hash),
		verificationData.Signature)
	if err != nil {
		return BadRequestResponse(c, errors.New(errVerifying))
	}

	if verified != true {
		return ForbiddenResponse(c, errors.New(signatureDidNotMatchResponse))
	}
	return nil
}

func hashRequestBody(reqBody interface{}, c *gin.Context) ([]byte, error) {
	var err error
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		err = fmt.Errorf(marshalError+" %v", err)
		return []byte{}, BadRequestResponse(c, err)
	}

	return utils.Hash(reqJSON), err
}

func returnAccountIfVerifiedFromParsedRequest(reqBody interface{}, verificationData verification, c *gin.Context) (models.Account, error) {
	if err := verifyParsedRequest(reqBody, verificationData, c); err != nil {
		return models.Account{}, err
	}

	return verificationData.getAccount(c)
}

func returnAccountIfVerifiedFromStringRequest(reqAsString string, dest interface{}, verificationData verification, c *gin.Context) (models.Account, error) {
	if err := verifyAndParseStringRequest(reqAsString, dest, verificationData, c); err != nil {
		return models.Account{}, err
	}

	return verificationData.getAccount(c)
}

func returnAccountIdWithParsedRequest(reqBody interface{}, verificationData verification, c *gin.Context) (string, error) {
	hash, err := hashRequestBody(reqBody, c)
	if err != nil {
		return "", err
	}

	return returnAccountId(hash, verificationData, c)
}

func returnAccountIdWithStringRequest(reqAsString string, verificationData verification, c *gin.Context) (string, error) {
	return returnAccountId(utils.Hash([]byte(reqAsString)), verificationData, c)
}

func returnAccountId(hash []byte, verificationData verification, c *gin.Context) (string, error) {
	if err := verifyRequest(hash, verificationData, c); err != nil {
		return "", err
	}

	return verificationData.getAccountId(c)
}

func getPermissionHash(publicKey, key string, c *gin.Context) (string, error) {
	permissionHash, err := utils.HashString(publicKey + key)
	if err != nil {
		return "", InternalErrorResponse(c, err)
	}
	return permissionHash, nil
}

func getPermissionHashKeyForBadger(metadataKey string) string {
	return metadataKey + "_permissionHash"
}

func verifyPermissions(publicKey, key, expectedPermissionHash string, c *gin.Context) error {
	if expectedPermissionHash == "" {
		return ForbiddenResponse(c, errors.New("resource is ineligible for modification"))
	}
	permissionHash, err := getPermissionHash(publicKey, key, c)
	if err != nil {
		return err
	}
	if permissionHash != expectedPermissionHash {
		return ForbiddenResponse(c, errors.New(notAuthorizedResponse))
	}
	return nil
}
