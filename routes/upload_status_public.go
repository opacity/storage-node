package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

// FileUploadCompletedPublicRes ...
type FileUploadCompletedPublicRes struct {
	Shortlink string `json:"shortlink"`
}

// CheckUploadStatusPublicHandler godoc
// @Summary check status of an upload
// @Description check status of an upload
// @Accept  json
// @Produce  json
// @Param UploadStatusReq body routes.UploadStatusReq true "an object to poll upload status"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 404 {string} string "file or account not found"
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

	return OkResponse(c, FileUploadCompletedPublicRes{
		Shortlink: publicShare.PublicID,
	})
}
