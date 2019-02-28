package routes

import (
	"fmt"

	"errors"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type uploadFileReq struct {
	AccountID string `json:"accountID" binding:"required,len=64"`
	UploadID  string `json:"uploadID" binding:"required"`
	FileData  string `json:"fileData" binding:"required"`
}

type uploadFileRes struct {
	Status string `json:"status"`
}

/*UploadFileHandler is a handler for the user to upload files*/
func UploadFileHandler() gin.HandlerFunc {
	return gin.HandlerFunc(uploadFile)
}

func uploadFile(c *gin.Context) {
	request := uploadFileReq{}
	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		BadRequest(c, err)
		return
	}

	account, err := models.GetAccountById(request.AccountID)
	if err != nil {
		NotFound(c, errors.New("no account with id: "+request.AccountID))
		return
	}

	objectKey := fmt.Sprintf("%s%s", account.S3Prefix(), request.UploadID)
	if err := utils.SetDefaultBucketObject(objectKey, request.FileData); err != nil {
		InternalError(c, err)
		return
	}

	OkResponse(c, uploadFileRes{
		Status: "File is uploaded",
	})
}
