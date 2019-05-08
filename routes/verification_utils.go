package routes

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"strings"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

const signatureDidNotMatchResponse = "signature did not match"
const errVerifying = "error verifying signature"
const marshalError = "bad request, unable to marshal request body: "

type verification struct {
	// signature without 0x prefix is broken into
	// V: sig[0:63]
	// R: sig[64:127]
	// S: sig[128:129]
	Signature string `json:"signature" binding:"required,len=130" minLength:"130" maxLength:"130" example:"a 130 character string created when you signed the request with your private key or account handle"`
	Address   string `json:"address" binding:"required,len=42" minLength:"42" maxLength:"42" example:"a 42-char eth address with 0x prefix"`
}

func verifyRequest_v2(reqBody string, dest interface{}, address string, signature string, c *gin.Context) error {
	var err error
	hash, err := hashRequestBody_v2(reqBody, c)

	verified, err := utils.VerifyFromStrings(address, hex.EncodeToString(hash),
		signature)
	if err != nil {
		BadRequestResponse(c, errors.New(errVerifying))
		return err
	}

	if verified != true {
		err = errors.New(signatureDidNotMatchResponse)
		ForbiddenResponse(c, err)
		return err
	}

	if err := utils.ParseRequestBody_v2(reqBody, dest); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body: %v", err)
		return err
	}

	return nil
}

func verifyRequest(reqBody interface{}, address string, signature string, c *gin.Context) error {
	var err error
	hash, err := hashRequestBody(reqBody, c)

	verified, err := utils.VerifyFromStrings(address, hex.EncodeToString(hash),
		signature)
	if err != nil {
		BadRequestResponse(c, errors.New(errVerifying))
		return err
	}

	if verified != true {
		err = errors.New(signatureDidNotMatchResponse)
		ForbiddenResponse(c, err)
		return err
	}

	return nil
}

func hashRequestBody(reqBody interface{}, c *gin.Context) ([]byte, error) {
	var err error
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		err = fmt.Errorf(marshalError+" %v", err)
		BadRequestResponse(c, err)
		return []byte{}, err
	}

	return utils.Hash(reqJSON), nil
}

func hashRequestBody_v2(reqBody string, c *gin.Context) ([]byte, error) {
	return utils.Hash([]byte(reqBody)), nil
}

func returnAccountIfVerified(reqBody interface{}, address string, signature string, c *gin.Context) (models.Account, error) {
	var account models.Account
	if err := verifyRequest(reqBody, address, signature, c); err != nil {
		return account, err
	}

	accountID := strings.TrimPrefix(address, "0x")

	// validate user
	account, err := models.GetAccountById(accountID)
	if err != nil || len(account.AccountID) == 0 {
		AccountNotFoundResponse(c, accountID)
		return account, err
	}

	return account, err
}

func returnAccountID(reqBody interface{}, signature string, c *gin.Context) (string, error) {
	hash, err := hashRequestBody(reqBody, c)
	if err != nil {
		BadRequestResponse(c, err)
		return "", err
	}

	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		BadRequestResponse(c, err)
		return "", err
	}

	publicKey, err := utils.Recover(hash, sigBytes)
	if err != nil {
		BadRequestResponse(c, err)
		return "", err
	}
	return strings.TrimPrefix(utils.PubkeyToAddress(*publicKey).String(), "0x"), nil
}
