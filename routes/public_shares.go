package routes

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type publicShareOpsReq struct {
	verification
	requestBody
	publicShareObj publicShareObj
}

type publicShareObj struct {
	Shortlink string `json:"shortlink" binding:"required" example:"the short link of the completed file"`
}

type shortlinkFileResp struct {
	URL string `json:"url"`
}

type viewsCountResp struct {
	Count int `json:"count"`
}

func (v *publicShareOpsReq) getObjectRef() interface{} {
	return &v.publicShareObj
}

// ShortlinkFileHandler godoc
// @Summary get S3 url for a file
// @Description get the S3 URL for a publicly shared file
// @Accept  json
// @Produce  json
// @Success 200 {object} shortlinkFileResp
// @Failure 400 {string} string "bad request, unable to update views count, with the error"
// @Failure 404 {string} string "file does not exist"
// @Router /api/v2/public-share/:shortlink [get]
/*ShortlinkFileHandler is a handler for the user get the S3 url of a public file*/
func ShortlinkFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(shortlinkURL)
}

// ViewsCountHandler godoc
// @Summary get views count
// @Description get the views count for a publicly shared file
// @Accept json
// @Produce json
// @description requestBody should be a stringified version of):
// @description {
// @description 	"shortlink": "the shortlink of the completed file",
// @description }
// @Success 200 {object} viewsCountResp
// @Failure 400 {string} string "bad request, unable to get views count"
// @Failure 404 {string} string "file does not exist"
// @Router /api/v2/public-share/views-count [post]
/*ViewsCountHandler is a handler for the user get the views count a public file*/
func ViewsCountHandler() gin.HandlerFunc {
	return ginHandlerFunc(viewsCount)
}

// RevokePublicShareHandler godoc
// @Summary revokes public share
// @Description remove a public share entry, revoke the share
// @Accept json
// @Produce json
// @description requestBody should be a stringified version of):
// @description {
// @description 	"shortlink": "the shortlink of the completed file",
// @description }
// @Success 200 {object} viewsCountResp
// @Failure 400 {string} string "bad request, unable to get views count"
// @Failure 404 {string} string "file does not exist"
// @Failure 500 {string} string "public file could not be deleted"
// @Router /api/v2/public-share/views-count [post]
/*RevokePublicShareHandler is a handler for the user get the views count a public file*/
func RevokePublicShareHandler() gin.HandlerFunc {
	return ginHandlerFunc(revokePublicShare)
}

func shortlinkURL(c *gin.Context) error {
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

func viewsCount(c *gin.Context) error {
	request := publicShareOpsReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	publicShare, err := models.GetPublicShareByID(request.publicShareObj.Shortlink)
	if err != nil {
		return NotFoundResponse(c, errors.New("public share does not exist"))
	}

	completedFile, err := models.GetCompletedFileByFileID(publicShare.FileID)
	if err != nil {
		return NotFoundResponse(c, errors.New("file does not exist"))
	}

	if err := verifyPermissions(request.PublicKey, publicShare.FileID, completedFile.ModifierHash, c); err != nil {
		return err
	}

	return OkResponse(c, viewsCountResp{
		Count: publicShare.ViewsCount,
	})
}

func revokePublicShare(c *gin.Context) error {
	request := publicShareOpsReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	publicShare, err := models.GetPublicShareByID(request.publicShareObj.Shortlink)
	if err != nil {
		return NotFoundResponse(c, errors.New("public share does not exist"))
	}

	if err := utils.DeleteDefaultBucketObjectKeys(models.GetFileDataPublicKey(publicShare.FileID)); err != nil {
		return InternalErrorResponse(c, err)
	}

	if err = publicShare.RemovePublicShare(); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, StatusRes{
		Status: "Public share revoked",
	})
}
