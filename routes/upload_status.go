package routes

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type UploadStatusObj struct {
	FileHandle string `form:"fileHandle" binding:"required,len=64" minLength:"64" maxLength:"64" example:"a deterministically created file handle"`
}

type uploadStatusReq struct {
	verification
	RequestBody string `json:"requestBody" binding:"required" example:"should produce routes.UploadStatusObj, see description for example"`
}

func GetUploadStatusHandler() gin.HandlerFunc {
	return ginHandlerFunc(getUploadStatus)
}

func getUploadStatus(c *gin.Context) {
	request := uploadStatusReq{}

	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		BadRequestResponse(c, fmt.Errorf("bad request, unable to parse request body:  %v", err))
		return
	}

	requestBodyParsed := UploadFileObj{}

	account, err := returnAccountIfVerifiedFromStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c)
	if err != nil {
		return
	}

	file, err := models.GetFileById(requestBodyParsed.FileHandle)
	if err != nil || len(file.FileID) == 0 {
		FileNotFoundResponse(c, requestBodyParsed.FileHandle)
		return
	}

	completedFile, err := file.FinishUpload()
	if err != nil {
		if err == models.IncompleteUploadErr {
			OkResponse(c, chunkUploadCompletedRes)
			return
		}
		InternalErrorResponse(c, err)
		return
	}

	if err := account.UseStorageSpaceInByte(int(completedFile.FileSizeInByte)); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	OkResponse(c, fileUploadCompletedRes)
}
