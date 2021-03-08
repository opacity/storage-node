package routes

import (
	"github.com/gin-gonic/gin"
)

// UploadFilePublicHandler godoc
// @Summary upload a chunk of a file
// @Description upload a chunk of a file. The first partIndex must be 1.
// @Accept mpfd
// @Produce json
// @Param UploadFileReq body routes.UploadFileReq true "an object to upload a chunk of a file"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
// @description 	"partIndex": 1,
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 403 {object} routes.accountCreateRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
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
