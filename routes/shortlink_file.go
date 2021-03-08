package routes

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type shortlinkFileResp struct {
	URL string `json:"url"`
}

// ShortlinkFileHandler godoc
// @Summary get S3 url for a file
// @Description get the S3 URL for a publicly shared file
// @Accept  json
// @Produce  json
// @Param shortlinkFileReq
// @Success 200 {object} shortlinkFileResp
// @Failure 400 {string} string "bad request, unable to update views count, with the error"
// @Failure 404 {string} string "file does not exist"
// @Router /api/v2/public-share/:shortlink [get]
/*ShortlinkFileHandler is a handler for the user get the S3 url of a public file*/
func ShortlinkFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(shortlinkFile)
}

func shortlinkFile(c *gin.Context) error {
	shortlink := c.Param("shortlink")
	publicShare, err := models.GetPublicShareByID(shortlink)

	fileDataPublicKey := models.GetFileDataPublicKey(publicShare.FileID)
	if err != nil || !utils.DoesDefaultBucketObjectExist(fileDataPublicKey) {
		return NotFoundResponse(c, errors.New("file does not exist"))
	}

	err = publicShare.UpdateViewsCount()
	if err != nil {
		return BadRequestResponse(c, err)
	}
	return OkResponse(c, shortlinkFileResp{
		URL: fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s", utils.Env.AwsRegion, utils.Env.BucketName, fileDataPublicKey),
	})
}
