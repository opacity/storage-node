package routes

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

// PublicShareOpsReq...
type PublicShareOpsReq struct {
	verification
	requestBody
	publicShareObj PublicShareObj
}

// PrivateToPublicReq...
type PrivateToPublicReq struct {
	verification
	requestBody
	privateToPublicObj PrivateToPublicObj
}

// PublicShareObj...
type PublicShareObj struct {
	Shortlink string `json:"shortlink" validate:"required" example:"the short link of the completed file"`
}

type PublicFileDownloadResp struct {
	S3URL          string `json:"s3_url"`
	S3ThumbnailURL string `json:"s3_thumbnail_url"`
}

type CreateShortlinkResp struct {
	ShortID string `json:"short_id"`
}

type CreateShortlinkReq struct {
	verification
	requestBody
	createShortlinkObj models.CreateShortlinkObj
}

type viewsCountResp struct {
	Count int `json:"count"`
}

func (v *PublicShareOpsReq) getObjectRef() interface{} {
	return &v.publicShareObj
}

func (v *CreateShortlinkReq) getObjectRef() interface{} {
	return &v.createShortlinkObj
}

// CreateShortlinkHandler godoc
// @Summary creates a shortlink
// @Description this endpoint will created a new shortlink based on the fileHandle, a title and a description
// @Accept json
// @Produce json
// @Param CreateShortlinkReq body routes.CreateShortlinkReq true "an object to create a shortlink for a public shared file"
// @description requestBody should be a stringified version of:
// @description {
// @description 	"fileId": "the ID of the file",
// @description 	"title": "the title of the file",
// @description 	"description": "a description of the file",
// @description 	"mimeType": "the file mimeType example: image/png",
// @description 	"fileExtension": "the file extension, example: png"
// @description }
// @Success 200 {object} routes.CreateShortlinkResp
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 403 {string} string "signature did not match"
// @Failure 404 {string} string "the data does not exist"
// @Router /api/v2/public-share/shortlink [post]
/*CreateShortlinkHandler is a handler to create a shortlink for a public shared file*/
func CreateShortlinkHandler() gin.HandlerFunc {
	return ginHandlerFunc(createShortLinkWithContext)
}

// ShortlinkFileHandler godoc
// @Summary get the url for a publicly shared file
// @Description get the the URL for a publicly shared file
// @Accept json
// @Produce json
// @Param shortlink path string true "shortlink ID"
// @Success 200 {object} PublicFileDownloadResp
// @Failure 404 {string} string "file does not exist"
// @Failure 500 {string} string "there was an error parsing your request"
// @Router /api/v2/public-share/:shortlink [get]
/*ShortlinkFileHandler is a handler for the user get the the url of a public file*/
func ShortlinkFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(shortlinkURL)
}

// ViewsCountHandler godoc
// @Summary get views count
// @Description get the views count for a publicly shared file
// @Accept json
// @Produce json
// @Param PublicShareOpsReq body routes.PublicShareOpsReq true "an object to do operations on a public share"
// @description requestBody should be a stringified version of:
// @description {
// @description 	"shortlink": "the shortlink of the completed file",
// @description }
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
// @Accept  json
// @Produce  json
// @Param PublicShareOpsReq body routes.PublicShareOpsReq true "an object to do operations on a public share"
// @description requestBody should be a stringified version of):
// @description {
// @description 	"shortlink": "the shortlink of the completed file",
// @description }
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

func createShortLinkWithContext(c *gin.Context) error {
	request := CreateShortlinkReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	storageType := account.PlanInfo.FileStorageType

	awsKey := models.GetFileDataPublicKey(request.createShortlinkObj.FileID)
	if !utils.DoesDefaultBucketObjectExist(awsKey, storageType) {
		return NotFoundResponse(c, errors.New("file does not exist"))
	}

	publicShare, err := models.CreatePublicShare(request.createShortlinkObj)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return NotFoundResponse(c, errors.New("the data does not exist"))
		}
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, CreateShortlinkResp{
		ShortID: publicShare.PublicID,
	})
}

func shortlinkURL(c *gin.Context) error {
	shortlink := c.Param("shortlink")
	publicShare, err := models.GetPublicShareByID(shortlink)

	fileDataPublicKey := models.GetFileDataPublicKey(publicShare.FileID)
	if err != nil || !utils.DoesDefaultBucketObjectExist(fileDataPublicKey, publicShare.FileStorageType) {
		return NotFoundResponse(c, errors.New("file does not exist"))
	}

	err = publicShare.UpdateViewsCount()
	if err != nil {
		return InternalErrorResponse(c, errors.New("there was an error parsing your request"))
	}

	fileURL, thumbnailURL := models.GetPublicFileDownloadData(publicShare.FileID, publicShare.FileStorageType)

	return OkResponse(c, PublicFileDownloadResp{
		S3URL:          fileURL,
		S3ThumbnailURL: thumbnailURL,
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

	_, err := request.getAccount(c)
	if err != nil {
		return err
	}

	// storageType := account.PlanInfo.FileStorageType

	publicShare, err := models.GetPublicShareByID(request.publicShareObj.Shortlink)
	if err != nil {
		return NotFoundResponse(c, errors.New("public share does not exist"))
	}

	utils.DeleteDefaultBucketObject(models.GetFileDataPublicKey(publicShare.FileID), publicShare.FileStorageType)
	utils.DeleteDefaultBucketObject(models.GetPublicThumbnailKey(publicShare.FileID), publicShare.FileStorageType)

	if err = publicShare.RemovePublicShare(); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, StatusRes{
		Status: "Public share revoked",
	})
}
