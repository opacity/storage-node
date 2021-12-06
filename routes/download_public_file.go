package routes

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type downloadPublicFileRes struct {
	FileDownloadUrl          string `json:"fileDownloadUrl" example:"a URL to use to download the public file"`
	FileDownloadThumbnailUrl string `json:"fileDownloadThumbnailUrl" example:"a URL to use to download the public file thumbnail"`
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

// @TODO: Is this really needed?
func downloadPublicFile(c *gin.Context) error {
	request := DownloadFileObj{}

	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		return BadRequestResponse(c, err)
	}

	completedFile, err := models.GetCompletedFileByFileID(request.FileID)
	if err != nil {
		return NotFoundResponse(c, err)
	}

	// @TODO: Remove default S3
	if !utils.DoesDefaultBucketObjectExist(models.GetFileDataPublicKey(request.FileID), completedFile.StorageType) {
		return NotFoundResponse(c, errors.New("such data does not exist"))
	}

	fileURL, thumbnailURL := models.GetPublicFileDownloadData(request.FileID, completedFile.StorageType)

	return OkResponse(c, downloadPublicFileRes{
		FileDownloadUrl:          fileURL,
		FileDownloadThumbnailUrl: thumbnailURL,
	})
}
