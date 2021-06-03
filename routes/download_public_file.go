package routes

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type downloadPublicFileRes struct {
	FileDownloadUrlPublic          string `json:"fileDownloadUrlPublic" example:"a URL to use to download the public file"`
	FileDownloadThumbnailUrlPublic string `json:"fileDownloadThumbnailUrlPublic" example:"a URL to use to download the public file thumbnail"`
}

// DownloadPublicFileHandler godoc
// @Summary returns the URLs for a public file and it's thumbnail
// @Description returns the URLs for a public file and it's thumbnail, if no thumbnail is present, return a default one
// @Param routes.DownloadFileObj body routes.DownloadFileObj true "download object for non-signed requests"
// @Accept json
// @Produce json
// @Success 200 {object} routes.downloadPublicFileRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 404 {string} string "such data does not exist"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v2/download/public [post]
/*DownloadPublicFileHandler returns the URLs for a public file and it's thumbnail, if no thumbnail is present, return a default one*/
func DownloadPublicFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(downloadPublicFile)
}

func downloadPublicFile(c *gin.Context) error {
	request := DownloadFileObj{}

	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		return BadRequestResponse(c, err)
	}

	if !utils.DoesDefaultBucketObjectExist(models.GetFileDataPublicKey(request.FileID)) {
		return NotFoundResponse(c, errors.New("such data does not exist"))
	}

	fileURL, thumbnailURL := models.GetPublicFileDownloadData(request.FileID)
	thumbnailURL = checkFileThumbnail(thumbnailURL)

	return OkResponse(c, downloadPublicFileRes{
		FileDownloadUrlPublic:          fileURL,
		FileDownloadThumbnailUrlPublic: thumbnailURL,
	})
}

func checkFileThumbnail(thumbnailURL string) string {
	resp, err := http.Head(thumbnailURL)

	if err == nil && resp.StatusCode == http.StatusOK {
		return thumbnailURL
	}

	return "https://s3.us-east-2.amazonaws.com/opacity-public/thumbnail_default.png"
}
