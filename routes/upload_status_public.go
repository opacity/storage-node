package routes

import (
	"bytes"

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

// CheckUploadStatusPublicHandler godoc
// @Summary check status of a public upload
// @Description check status of a public upload
// @Accept json
// @Produce json
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
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
	request := UploadStatusReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	fileID := request.uploadStatusObj.FileHandle
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

	if err := verifyPermissions(request.PublicKey, request.uploadStatusObj.FileHandle, file.ModifierHash, c); err != nil {
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

	if err := GeneratePublicThumbnail(completedFile.FileID); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, FileUploadCompletedPublicRes{
		Shortlink:    publicShare.PublicID,
		ThumbnailURL: models.GetPublicThumbnailKey(publicShare.PublicID),
	})
}

func GeneratePublicThumbnail(fileID string) error {
	thumbnailKey := models.GetPublicThumbnailKey(fileID)
	fileDataPublicKey := models.GetFileDataPublicKey(fileID)
	publicFileObj, err := utils.GetBucketObject(fileDataPublicKey, true)

	if err != nil {
		return err
	}

	image, err := imaging.Decode(publicFileObj)

	if err != nil {
		return err
	}

	thumbnailImage := imaging.Thumbnail(image, 1200, 628, imaging.Lanczos)
	distThumbnailWriter := bytes.NewBufferString("")
	err = imaging.Encode(distThumbnailWriter, thumbnailImage, imaging.JPEG)
	if err != nil {
		return err
	}

	distThumbnailString := distThumbnailWriter.String()
	err = utils.SetDefaultBucketObject(thumbnailKey, distThumbnailString)

	return err
}
