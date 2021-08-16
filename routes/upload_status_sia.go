package routes

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

// CheckUploadStatusSiaHandler godoc
// @Summary check status of a Sia upload
// @Description check status of a Sia upload
// @Accept json
// @Produce json
// @Param UploadStatusReq body routes.UploadStatusReq true "an object to poll upload status for a Sia file"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 403 {string} string "signature did not match"
// @Failure 404 {string} string "file or account not found"
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v2/sia/upload-status [post]
/*CheckUploadStatusSiaHandler is a handler for checking upload statuses*/
func CheckUploadStatusSiaHandler() gin.HandlerFunc {
	return ginHandlerFunc(checkUploadStatusSia)
}

func checkUploadStatusSia(c *gin.Context) error {
	request := UploadStatusReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	fileID := request.uploadStatusObj.FileHandle
	siaProgressFile, err := models.GetSiaProgressFileById(fileID)
	if err != nil {
		return SiaFileNotInitialised(c)
	}

	if err := verifyPermissions(request.PublicKey, fileID, siaProgressFile.ModifierHash, c); err != nil {
		return err
	}

	completedFile, completedErr := models.GetCompletedFileByFileID(fileID)
	if completedErr == nil && len(completedFile.FileID) != 0 {
		return OkResponse(c, fileUploadCompletedRes)
	}

	siaFileMetadata, siaFileMetadataErr := utils.GetSiaFileMetadataByPath(fileID)
	if err != nil {
		if strings.Contains(siaFileMetadataErr.Error(), "path does not exist") {
			return NotFoundResponse(c, errors.New("sia file was not uploaded"))
		}
		return InternalErrorResponse(c, err)
	}

	if siaFileMetadata.File.Available {
		completedFile := models.CompletedFile{
			FileID:         siaProgressFile.FileID,
			ExpiredAt:      siaProgressFile.ExpiredAt,
			FileSizeInByte: int64(siaFileMetadata.File.Filesize),
			ModifierHash:   siaProgressFile.ModifierHash,
			StorageType:    models.Sia,
		}
		if err := models.DB.Save(&completedFile).Error; err != nil {
			utils.DeleteSiaFile(completedFile.FileID)
			InternalErrorResponse(c, errors.New("error finishing sia upload"))
		}
	} else {
		return OkResponse(c, StatusRes{
			Status: "sia file still uploading",
		})
	}

	if err := account.UseStorageSpaceInByte(completedFile.FileSizeInByte); err != nil {
		errSia := utils.DeleteSiaFile(completedFile.FileID)
		errSql := models.DB.Delete(&completedFile).Error
		return InternalErrorResponse(c, utils.CollectErrors([]error{err, errSia, errSql}))
	}

	models.DB.Delete(&siaProgressFile)

	return OkResponse(c, fileUploadCompletedRes)
}
