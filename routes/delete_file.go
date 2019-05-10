package routes

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type deleteFileObj struct {
	FileID string `json:"fileID" binding:"required" example:"the handle of the file"`
}

type deleteFileReq struct {
	verification
	RequestBody string `json:"requestBody" binding:"required" example:"should produce routes.deleteFileObj, see description for example"`
}

type deleteFileRes struct {
}

// DeleteFileHandler godoc
// @Summary delete a file
// @Description delete a file
// @Accept  json
// @Produce  json
// @Param deleteFileReq body routes.deleteFileReq true "file deletion object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileID": "the handle of the file",
// @description }
// @Success 200 {object} routes.deleteFileRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/file [delete]
/*DeleteFileHandler is a handler for the user to upload files*/
func DeleteFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(deleteFile)
}

func deleteFile(c *gin.Context) {
	request := deleteFileReq{}

	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		BadRequestResponse(c, err)
		return
	}

	requestBodyParsed := deleteFileObj{}

	var account models.Account
	var err error
	if account, err = returnAccountIfVerifiedFromStringRequest(request.RequestBody, &requestBodyParsed, request.Address,
		request.Signature, c); err != nil {
		return
	}

	if err := utils.DeleteDefaultBucketObject(requestBodyParsed.FileID); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	var completedFile models.CompletedFile
	if completedFile, err = models.GetCompletedFileByFileID(requestBodyParsed.FileID); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	if err := account.UseStorageSpaceInByte(-1 * int(completedFile.FileSizeInByte)); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	if err := models.DB.Delete(&completedFile).Error; err != nil {
		InternalErrorResponse(c, err)
		return
	}

	OkResponse(c, deleteFileRes{})
}
