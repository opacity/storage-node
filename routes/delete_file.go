package routes

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type deleteFileReq struct {
	AccountID string `binding:"required,len=64"`
	FileID    string `binding:"required"`
}

type deleteFileRes struct {
}

/*DeleteFileHandler is a handler for the user to upload files*/
func DeleteFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(deleteFile)
}

func deleteFile(c *gin.Context) {
	request := deleteFileReq{
		AccountID: c.Param("accountID"),
		FileID:    c.Param("fileID"),
	}
	if err := utils.Validator.Struct(request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		BadRequestResponse(c, err)
		return
	}

	// validate user
	account, err := models.GetAccountById(request.AccountID)
	if err != nil || len(account.AccountID) == 0 {
		AccountNotFoundResponse(c, request.AccountID)
		return
	}

	if err := utils.DeleteDefaultBucketObject(request.FileID); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	OkResponse(c, deleteFileRes{})
}
