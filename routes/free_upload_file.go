package routes

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

type freeUploadFileReq struct {
	UploadID string `json:"uploadID" validate:"required"`
	FileData string `json:"fileData" validate:"required"`
}

type freeUploadFileRes struct {
	Status string `json:"status"`
}

func FreeUploadFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(freeUploadFile)
}

func freeUploadFile(c *gin.Context) error {
	request := freeUploadFileReq{}
	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		return BadRequestResponse(c, err)
	}

	objectKey := fmt.Sprintf("%s%s", "free_upload/", request.UploadID)

	if err := utils.SetDefaultBucketObject(objectKey, request.FileData); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, StatusRes{
		Status: "File is uploaded",
	})
}
