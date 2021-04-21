package routes

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type PublicShareOpsReq struct {
	verification
	requestBody
	publicShareObj PublicShareObj
}

type PublicShareObj struct {
	Shortlink string `json:"shortlink" validate:"required" example:"the short link of the completed file"`
}

type shortlinkFileResp struct {
	URL string `json:"url"`
}

type viewsCountResp struct {
	Count int `json:"count"`
}

func (v *PublicShareOpsReq) getObjectRef() interface{} {
	return &v.publicShareObj
}

// ShortlinkFileHandler godoc
// @Summary get S3 url for a publicly shared file
// @Description get the S3 URL for a publicly shared file
// @Accept  json
// @Produce  json
// @Param shortlink path string true "shortlink ID"
// @Success 200 {object} shortlinkFileResp
// @Failure 404 {string} string "file does not exist"
// @Failure 500 {string} string "there was an error parsing your request"
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
// @description requestBody should be a stringified version of:
// @description {
// @description 	"shortlink": "the shortlink of the completed file",
// @description }
// @Param publicShareOpsReq body publicShareOpsReq true "an object to do operations on a public share"
// @Success 200 {object} routes.viewsCountResp
// @Failure 400 {string} string "bad request, unable to get views count"
// @Failure 403 {string} string "signature did not match"
// @Failure 404 {string} string "public share or file does not exist"
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
// @Param publicShareOpsReq body publicShareOpsReq true "an object to do operations on a public share"
// @Success 200 {object} routes.StatusRes
// @Failure 400 {string} string "bad request, unable to revoke public share"
// @Failure 403 {string} string "signature did not match"
// @Failure 404 {string} string "file does not exist"
// @Failure 500 {string} string "public file could not be deleted from databse or S3"
// @Router /api/v2/public-share/revoke [post]
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
		return InternalErrorResponse(c, errors.New("there was an error parsing your request"))
	}
	return OkResponse(c, shortlinkFileResp{
		URL: fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s", utils.Env.AwsRegion, utils.Env.BucketName, fileDataPublicKey),
	})
}

func viewsCount(c *gin.Context) error {
	request := PublicShareOpsReq{}

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
	request := PublicShareOpsReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	publicShare, err := models.GetPublicShareByID(request.publicShareObj.Shortlink)
	if err != nil {
		return NotFoundResponse(c, errors.New("public share does not exist"))
	}

	utils.DeleteDefaultBucketObjectKeys(models.GetFileDataPublicKey(publicShare.FileID))

	if err = publicShare.RemovePublicShare(); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, StatusRes{
		Status: "Public share revoked",
	})
}
