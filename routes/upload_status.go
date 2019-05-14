package routes

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type UploadStatusObj struct {
	FileHandle string `json:"fileHandle" binding:"required,len=64" minLength:"64" maxLength:"64" example:"a deterministically created file handle"`
}

type uploadStatusReq struct {
	verification
	RequestBody string `json:"requestBody" binding:"required" example:"should produce routes.UploadStatusObj, see description for example"`
}

// CheckUploadStatusHandler godoc
// @Summary check status of an upload
// @Description check status of an upload
// @Accept  json
// @Produce  json
// @Param uploadStatusReq body routes.uploadStatusReq true "an object to poll upload status"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
// @description }
// @Success 200 {object} routes.uploadFileRes
// @Failure 404 {string} string "file or account not found"
// @Failure 403 {string} string "signature did not match"
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/upload-status [post]
/*CheckUploadStatusHandler is a handler for checking upload statuses*/
func CheckUploadStatusHandler() gin.HandlerFunc {
	return ginHandlerFunc(checkUploadStatus)
}

func checkUploadStatus(c *gin.Context) {
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
