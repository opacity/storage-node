package routes

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type downloadSimpleFileObj struct {
	FileID string `json:"fileID" binding:"required" example:"the handle of the file"`
}

// DownloadSimpleFileHandler godoc
// @Summary download a file without cryptographic verification
// @Description download a file without cryptographic verification
// @Accept  json
// @Produce  json
// @Param downloadSimpleFileObj body routes.downloadSimpleFileObj true "download object for non-signed requests"
// @Success 200 {object} routes.downloadFileRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 404 {string} string "such data does not exist"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/download [post]
/*DownloadSimpleFileHandler handles the downloading of a file without cryptographic verification*/
func DownloadSimpleFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(downloadSimpleFile)
}

func downloadSimpleFile(c *gin.Context) error {
	request := downloadSimpleFileObj{}

	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		return BadRequestResponse(c, err)
	}
	// verify object existed in S3
	if !utils.DoesDefaultBucketObjectExist(models.GetFileDataKey(request.FileID)) {
		return NotFoundResponse(c, errors.New("such data does not exist"))
	}

	if err := utils.SetDefaultObjectCannedAcl(models.GetFileDataKey(request.FileID), utils.CannedAcl_PublicRead); err != nil {
		return InternalErrorResponse(c, err)
	}

	if err := utils.SetDefaultObjectCannedAcl(models.GetFileMetadataKey(request.FileID), utils.CannedAcl_PublicRead); err != nil {
		return InternalErrorResponse(c, err)
	}

	url := fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s", utils.Env.AwsRegion, utils.Env.BucketName,
		request.FileID)

	return OkResponse(c, downloadFileRes{
		// Redirect to a different URL that client would have authorization to download it.
		FileDownloadUrl: url,
	})
}
