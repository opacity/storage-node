package routes

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

type freeUploadFileReq struct {
	UploadID string `json:"uploadID" binding:"required"`
	FileData string `json:"fileData" binding:"required"`
}

type freeUploadFileRes struct {
	Status string `json:"status"`
}

func FreeUploadFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(freeUploadFile)
}

func freeUploadFile(c *gin.Context) {
	request := freeUploadFileReq{}
	if err := utils.ParseRequestBody(c, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		BadRequestResponse(c, err)
		return
	}

	objectKey := fmt.Sprintf("%s%s", "free_upload/", request.UploadID)

	if err := utils.SetDefaultBucketObject(objectKey, request.FileData); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	OkResponse(c, uploadFileRes{
		Status: "File is uploaded",
	})
}
