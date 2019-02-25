package routes

import (
	"fmt"

	"github.com/gin-gonic/gin"
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

	OkResponse(c, uploadFileRes{
		Status: "stub for uploading a file with an existing subscription",
	})
}
