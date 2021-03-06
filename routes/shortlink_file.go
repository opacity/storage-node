package routes

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type shortlinkFileReq struct {
	Shortlink string `json:"shortlink"`
	FileName  string `json:"fileName"`
}

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
// @Failure 404 {string} string "file does not exist"
// @Router /api/v2/public-share/:shortlink [post]
/*ShortlinkFileHandler is a handler for the user get the S3 url of a public file*/
func ShortlinkFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(shortlinkFile)
}

func shortlinkFile(c *gin.Context) error {
	s := shortlinkFileReq{}
	if c.BindJSON(&s) == nil {
		s.Shortlink = c.Param("shortlink")
		file, err := models.GetFileByShortlink(s.Shortlink)
		if err != nil || !utils.DoesDefaultBucketObjectExist(models.GetFileNameKey(file.FileID, s.FileName)) {
			return NotFoundResponse(c, errors.New("file does not exist"))
		}

		return OkResponse(c, shortlinkFileResp{
			URL: fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s/%s", utils.Env.AwsRegion, utils.Env.BucketName, file.FileID, s.FileName),
		})
	}

	return BadRequestResponse(c, errors.New("error getting file"))
}
