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
	msg string `json:"msg"`
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
	if err != nil {
		AccountNotFoundResponse(c, request.AccountID)
		return
	}

	objectKey := account.GetS3ObjectKeyForFileID(request.FileID)
	if err := utils.DeleteDefaultBucketObject(objectKey); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	OkResponse(c, deleteFileRes{})
}
