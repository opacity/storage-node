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

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

const signatureDidNotMatchResponse = "signature did not match"
const errVerifying = "error verifying signature"
const marshalError = "bad request, unable to marshal request body: "

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

func (v verification) getAccount(c *gin.Context) (models.Account, error) {
	return returnAccountIfVerified(v.PublicKey, c)
}

type requestBody struct {
	RequestBody string `form:"requestBody" binding:"required" example:"should produce routes.InitFileUploadObj as example"`
}

func (v requestBody) getObjectAsString() string {
	return v.RequestBody
}

func verifyAndParseFormRequest(dest interface{}, c *gin.Context) error {
	defer c.Request.Body.Close()
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxRequestSize)
	err := c.Request.ParseMultipartForm(MaxRequestSize)
	if err != nil {
		BadRequestResponse(c, err)
		return err
	}

	t := reflect.TypeOf(dest)
	v := reflect.ValueOf(&dest).Elem()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i) // Get the field, returns https://golang.org/pkg/reflect/#StructField
		formTag := field.Tag.Get("form")
		fileTag := field.Tag.Get("formFile")

		if formTag == "" && fileTag == "" {
			continue
		}

		strV := ""
		if formTag != "" {
			strV = c.Request.FormValue(formTag)
		}

		if fileTag != "" {
			strV, err := readFileFromForm(fileTag, c)
			if err != nil {
				return err
			}
		}

		v.Field(i).SetString(strV)
	}
	if i, ok := dest.(verificationInterface); ok {
		if ii, ok := dest.(parsableObjectInterface); ok {
			return verifyAndParseStringRequest(ii.getObjectAsString(), ii.getObjectRef(), i.getVerification(), c)
		}
	}
	return nil
}

func readFileFromForm(fileTag string, c *gin.Context) (string, error) {
	multiFile, _, err := c.Request.FormFile(fileTag)
	defer multiFile.Close()
	if err != nil {
		InternalErrorResponse(c, err)
		return "", err
	}
	var fileBytes bytes.Buffer
	_, err = io.Copy(&fileBytes, multiFile)
	if err != nil {
		InternalErrorResponse(c, err)
		return "", err
	}
	return fileBytes.String(), nil
}

func verifyAndParseStringRequest(reqAsString string, dest interface{}, verificationData verification, c *gin.Context) error {
	hash := utils.Hash([]byte(reqAsString))

	if err := verifyRequest(hash, verificationData.PublicKey, verificationData.Signature, c); err != nil {
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

	if err := verifyRequest(hash, verificationData.PublicKey, verificationData.Signature, c); err != nil {
		return err
	}

	return nil
}

func verifyRequest(hash []byte, publicKey string, signature string, c *gin.Context) error {
	verified, err := utils.VerifyFromStrings(publicKey, hex.EncodeToString(hash),
		signature)
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

	return returnAccountIfVerified(verificationData.PublicKey, c)
}

func returnAccountIfVerifiedFromStringRequest(reqAsString string, dest interface{}, verificationData verification, c *gin.Context) (models.Account, error) {
	if err := verifyAndParseStringRequest(reqAsString, dest, verificationData, c); err != nil {
		return models.Account{}, err
	}

	return returnAccountIfVerified(verificationData.PublicKey, c)
}

func returnAccountIfVerified(publicKey string, c *gin.Context) (models.Account, error) {
	accountID, err := getAccountIdFromPublicKey(publicKey, c)
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
	if err := verifyRequest(hash, verificationData.PublicKey, verificationData.Signature, c); err != nil {
		return "", err
	}

	return getAccountIdFromPublicKey(verificationData.PublicKey, c)
}

func getAccountIdFromPublicKey(publicKey string, c *gin.Context) (string, error) {
	accountID, err := utils.HashString(publicKey)
	if err != nil {
		return "", InternalErrorResponse(c, err)
	}
	return accountID, err
}
