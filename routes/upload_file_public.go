package routes

import (
	"github.com/gin-gonic/gin"
)

// UploadFilePublicHandler godoc
// @Summary upload a chunk of a file
// @Description upload a chunk of a file. The first partIndex must be 1. The storage for this file does not count
// @Accept mpfd
// @Produce json
// @Param UploadFileReq body routes.UploadFileReq true "an object to upload a chunk of an unencrypted file (the storage for this file does no count)"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
// @description 	"partIndex": 1,
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 400 {string} string "bad request, unable to parse request body"
// @Failure 403 {string} string "signature did not match"
// @Failure 404 {string} string "file or account not found"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v2/upload-public [post]
/*UploadFileHandler is a handler for the user to upload files*/
func UploadFilePublicHandler() gin.HandlerFunc {
	return ginHandlerFunc(uploadFilePublic)
}

func uploadFilePublic(c *gin.Context) error {
	defer c.Request.Body.Close()

	request := UploadFileReq{}

	if err := verifyAndParseFormRequest(&request, c); err != nil {
		return err
	}

	return uploadChunk(request, c)
}
