package routes

import (
	"fmt"

	"strings"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type deleteFileObj struct {
	FileID string `binding:"required"`
}

type deleteFileReq struct {
	verification
	DeleteFile deleteFileObj `json:"deleteFile" binding:"required"`
}

type deleteFileRes struct {
}

/*DeleteFileHandler is a handler for the user to upload files*/
func DeleteFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(deleteFile)
}

func deleteFile(c *gin.Context) {
	request := deleteFileReq{}

	if err := utils.Validator.Struct(request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		BadRequestResponse(c, err)
		return
	}

	if err := verifyRequest(request.DeleteFile, request.Address, request.Signature, c); err != nil {
		return
	}

	accountID := strings.TrimPrefix(request.Address, "0x")

	// validate user
	account, err := models.GetAccountById(accountID)
	if err != nil || len(account.AccountID) == 0 {
		AccountNotFoundResponse(c, accountID)
		return
	}

	if err := utils.DeleteDefaultBucketObject(request.DeleteFile.FileID); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	OkResponse(c, deleteFileRes{})
}
