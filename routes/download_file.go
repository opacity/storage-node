package routes

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type downloadFileObj struct {
	FileID string `json:"fileID" binding:"required" example:"the handle of the file"`
}

type downloadFileRes struct {
	// Url should point to S3, thus client does not need to download it from this node.
	FileDownloadUrl string `json:"fileDownloadUrl" example:"a URL to use to download the file"`
	// Add other auth-token and expired within a certain period of time.
}

// DownloadFileHandler godoc
// @Summary download a file without cryptographic verification
// @Description download a file without cryptographic verification
// @Accept  json
// @Produce  json
// @Param downloadFileObj body routes.downloadFileObj true "download object for non-signed requests"
// @Success 200 {object} routes.downloadFileRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 404 {string} string "such data does not exist"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/download [post]
/*DownloadFileHandler handles the downloading of a file without cryptographic verification*/
func DownloadFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(downloadFile)
}

func downloadFile(c *gin.Context) error {
	request := downloadFileObj{}

	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		return BadRequestResponse(c, err)
	}

	url, err := GetFileDownloadURL(request.FileID)
	if err != nil {
		if err.Error() == "such data does not exist" {
			return NotFoundResponse(c, err)
		}
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, downloadFileRes{
		// Redirect to a different URL that client would have authorization to download it.
		FileDownloadUrl: url,
	})
}

func GetFileDownloadURL(fileID string) (string, error) {
	// verify object existed in S3
	if !utils.DoesDefaultBucketObjectExist(models.GetFileDataKey(fileID)) {
		return "", errors.New("such data does not exist")
	}

	if err := utils.SetDefaultObjectCannedAcl(models.GetFileDataKey(fileID), utils.CannedAcl_PublicRead); err != nil {
		return "", err
	}

	if err := utils.SetDefaultObjectCannedAcl(models.GetFileMetadataKey(fileID), utils.CannedAcl_PublicRead); err != nil {
		return "", err
	}
	url := fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s", utils.Env.AwsRegion, utils.Env.BucketName,
		fileID)

	return url, nil
}
