package routes

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

type deleteFileObj struct {
	FileID string `json:"fileID" binding:"required" example:"the handle of the file"`
}

type deleteFileReq struct {
	verification
	DeleteFile deleteFileObj `json:"deleteFile" binding:"required"`
}

type deleteFileRes struct {
}

// DeleteFileHandler godoc
// @Summary delete a file
// @Description delete a file
// @Accept  json
// @Produce  json
// @Param deleteFileReq body routes.deleteFileReq true "file deletion object"
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

	if err := utils.Validator.Struct(request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		BadRequestResponse(c, err)
		return
	}

	if _, err := returnAccountIfVerified(request.DeleteFile, request.Address, request.Signature, c); err != nil {
		return
	}

	if err := utils.DeleteDefaultBucketObject(request.DeleteFile.FileID); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	OkResponse(c, deleteFileRes{})
}
