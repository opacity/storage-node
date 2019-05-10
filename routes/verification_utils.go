package routes

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

const signatureDidNotMatchResponse = "signature did not match"
const errVerifying = "error verifying signature"
const marshalError = "bad request, unable to marshal request body: "

type verification struct {
	// signature without 0x prefix is broken into
	// R: sig[0:63]
	// S: sig[64:127]
	Signature string `json:"signature" binding:"required,len=128" minLength:"128" maxLength:"128" example:"a 128 character string created when you signed the request with your private key or account handle"`
	PublicKey string `json:"publicKey" binding:"required,len=66" minLength:"66" maxLength:"66" example:"a 66-character public key"`
}

func verifyAndParseStringRequest(reqAsString string, dest interface{}, verificationData verification, c *gin.Context) error {
	hash := utils.Hash([]byte(reqAsString))

	if err := verifyRequest(hash, verificationData.PublicKey, verificationData.Signature, c); err != nil {
		return err
	}

	if err := utils.ParseStringifiedRequest(reqAsString, dest); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body: %v", err)
		BadRequestResponse(c, err)
		return err
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
	accountID, err := utils.HashString(publicKey)
	if err != nil {
		InternalErrorResponse(c, err)
		return models.Account{}, err
	}

	// validate user
	account, err := models.GetAccountById(accountID)
	if err != nil || len(account.AccountID) == 0 {
		AccountNotFoundResponse(c, accountID)
		return account, err
	}

	return account, err
}

func returnAccountIdWithParsedRequest(reqBody interface{}, signature string, c *gin.Context) (string, error) {
	hash, err := hashRequestBody(reqBody, c)
	if err != nil {
		BadRequestResponse(c, err)
		return "", err
	}

	return returnAccountId(hash, signature, c)
}

func returnAccountIdWithStringRequest(reqAsString string, signature string, c *gin.Context) (string, error) {
	return returnAccountId(utils.Hash([]byte(reqAsString)), signature, c)
}

func returnAccountId(hash []byte, signature string, c *gin.Context) (string, error) {
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

	accountID, err := utils.HashString(utils.PubkeyToHex(*publicKey))
	if err != nil {
		InternalErrorResponse(c, err)
		return "", err
	}
	return accountID, nil
}
