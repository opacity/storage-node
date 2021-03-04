package routes

import "github.com/gin-gonic/gin"

// CheckUploadStatusPublicHandler godoc
// @Summary check status of an upload
// @Description check status of an upload
// @Accept  json
// @Produce  json
// @Param UploadStatusReq body routes.UploadStatusReq true "an object to poll upload status"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 404 {string} string "file or account not found"
// @Failure 403 {string} string "signature did not match"
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v2/upload-status-public [post]
/*CheckUploadStatusPublicHandler is a handler for checking upload statuses*/
func CheckUploadStatusPublicHandler() gin.HandlerFunc {
	return ginHandlerFunc(checkUploadStatusInit)
}

func checkUploadStatusPublicInit(c *gin.Context) error {
	err := checkUploadStatus(c, true)
	if err != nil {
		return err
	}

	return nil
}
