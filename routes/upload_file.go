package routes

import (
	"fmt"

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
	Status string
}

func UploadFileHandler() gin.HandlerFunc {
	return gin.HandlerFunc(uploadFile)
}

func uploadFile(c *gin.Context) {
	request := uploadFileReq{}
	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		BadRequest(c, fmt.Errorf("bad request, unable to parse request body:  %v", err))
		return
	}

	account, err := models.GetAccountById(request.AccountID)
	if err != nil {
		BadRequest(c, err)
		return
	}

	err = account.UpdateStorageUsedInByte(0)
	if err != nil {
		BadRequest(c, fmt.Errorf("Unable to update storage usage: %v", err))
		return
	}
	OkResponse(c, uploadFileRes{
		Status: "Stub for uploading file to S3",
	})
}
