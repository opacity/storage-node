package routes

import (
	"bytes"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

// FileUploadCompletedPublicRes ...
type FileUploadCompletedPublicRes struct {
	Shortlink    string `json:"shortlink"`
	ThumbnailURL string `json:"thumbnailUrl"`
}

type UploadStatusPublicObj struct {
	FileHandle string `json:"fileHandle" binding:"required,len=64" minLength:"64" maxLength:"64" example:"a deterministically created file handle"`
	MimeType   string `json:"mimeType" example:"image/jpeg"`
}

type UploadStatusPublicReq struct {
	verification
	requestBody
	uploadStatusPublicObj UploadStatusPublicObj
}

// CheckUploadStatusPublicHandler godoc
// @Summary check status of a public upload
// @Description check status of a public upload and creates a thumbnail in case the file is an image
// @Description "jpg" (or "jpeg"), "png", "gif", "tif" (or "tiff") and "bmp" are supported
// @Accept json
// @Produce json
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
// @description   "mimeType": "the mime type of the file"
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 404 {string} string "file not found"
// @Failure 403 {string} string "signature did not match"
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v2/upload-status-public [post]
/*CheckUploadStatusPublicHandler is a handler for checking upload statuses*/
func CheckUploadStatusPublicHandler() gin.HandlerFunc {
	return ginHandlerFunc(checkUploadStatusPublic)
}

func checkUploadStatusPublic(c *gin.Context) error {
	request := UploadStatusPublicReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	fileID := request.uploadStatusPublicObj.FileHandle
	completedFile, completedErr := models.GetCompletedFileByFileID(fileID)
	if completedErr == nil && len(completedFile.FileID) != 0 {
		if utils.DoesDefaultBucketObjectExist(models.GetFileDataPublicKey(fileID)) {
			return OkResponse(c, fileUploadCompletedRes)
		}
	}

	file, err := models.GetFileById(fileID)
	if err != nil || len(file.FileID) == 0 {
		return FileNotFoundResponse(c, fileID)
	}

	if err := verifyPermissions(request.PublicKey, request.uploadStatusPublicObj.FileHandle, file.ModifierHash, c); err != nil {
		return err
	}

	publicShare, err := file.FinishUploadPublic()
	if err != nil {
		if err == models.IncompleteUploadErr {
			incompleteIndexes, err := models.GetIncompleteIndexesAsArray(file.FileID, file.EndIndex)
			if err != nil || len(incompleteIndexes) == 0 {
				// fall back to the old way to get data
				incompleteIndexes = file.GetIncompleteIndexesAsArray()
			}
			return OkResponse(c, missingChunksRes{
				Status:         "chunks missing",
				MissingIndexes: incompleteIndexes,
				EndIndex:       file.EndIndex,
			})
		}
		return InternalErrorResponse(c, err)
	}

	if completedFile, err = models.GetCompletedFileByFileID(completedFile.FileID); err != nil {
		return InternalErrorResponse(c, err)
	}

	if err := utils.SetDefaultObjectCannedAcl(models.GetFileDataPublicKey(completedFile.FileID), utils.CannedAcl_PublicRead); err != nil {
		return InternalErrorResponse(c, err)
	}

	if err := GeneratePublicThumbnail(completedFile.FileID, request.uploadStatusPublicObj.MimeType); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, FileUploadCompletedPublicRes{
		Shortlink:    publicShare.PublicID,
		ThumbnailURL: models.GetPublicThumbnailKey(publicShare.PublicID),
	})
}

func GeneratePublicThumbnail(fileID string, mimeType string) error {
	thumbnailKey := models.GetPublicThumbnailKey(fileID)
	fileDataPublicKey := models.GetFileDataPublicKey(fileID)
	publicFileObj, err := utils.GetBucketObject(fileDataPublicKey, true)
	if err != nil {
		return err
	}
	defer publicFileObj.Close()

	image, err := imaging.Decode(publicFileObj)
	if err != nil {
		return err
	}

	_, extension := splitMime(mimeType)
	thumbnailFormat, _ := imaging.FormatFromExtension(extension)
	thumbnailImage := imaging.Thumbnail(image, 1200, 628, imaging.CatmullRom)
	distThumbnailWriter := new(bytes.Buffer)
	if err = imaging.Encode(distThumbnailWriter, thumbnailImage, thumbnailFormat); err != nil {
		return err
	}

	distThumbnailString := distThumbnailWriter.String()

	return utils.SetDefaultBucketObject(thumbnailKey, distThumbnailString)
}

func splitMime(s string) (string, string) {
	x := strings.Split(s, "/")
	if len(x) > 1 {
		return x[0], x[1]
	}
	return x[0], ""
}
