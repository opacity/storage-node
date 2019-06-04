package routes

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type UploadStatusObj struct {
	FileHandle string `json:"fileHandle" binding:"required,len=64" minLength:"64" maxLength:"64" example:"a deterministically created file handle"`
}

type UploadStatusReq struct {
	verification
	RequestBody string `json:"requestBody" binding:"required" example:"should produce routes.UploadStatusObj, see description for example"`
}

type missingChunksRes struct {
	Status         string  `json:"status" example:"chunks missing"`
	MissingIndexes []int64 `json:"missingIndexes" example:"[5, 7, 12]"`
	EndIndex       int     `json:"endIndex" example:"2"`
}

// CheckUploadStatusHandler godoc
// @Summary check status of an upload
// @Description check status of an upload
// @Accept  json
// @Produce  json
// @Param UploadStatusReq body routes.UploadStatusReq true "an object to poll upload status"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
// @description }
// @Success 200 {object} routes.uploadFileRes
// @Failure 404 {string} string "file or account not found"
// @Failure 403 {string} string "signature did not match"
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/upload-status [post]
/*CheckUploadStatusHandler is a handler for checking upload statuses*/
func CheckUploadStatusHandler() gin.HandlerFunc {
	return ginHandlerFunc(checkUploadStatus)
}

func checkUploadStatus(c *gin.Context) error {
	request := UploadStatusReq{}

	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		return BadRequestResponse(c, fmt.Errorf("bad request, unable to parse request body:  %v", err))
	}

	requestBodyParsed := UploadStatusObj{}

	account, err := returnAccountIfVerifiedFromStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c)
	if err != nil {
		return err
	}

	completedFile, completedErr := models.GetCompletedFileByFileID(requestBodyParsed.FileHandle)
	if completedErr == nil && len(completedFile.FileID) != 0 &&
		utils.DoesDefaultBucketObjectExist(models.GetFileDataKey(requestBodyParsed.FileHandle)) {
		return OkResponse(c, fileUploadCompletedRes)
	}

	file, err := models.GetFileById(requestBodyParsed.FileHandle)
	if err != nil || len(file.FileID) == 0 {
		return FileNotFoundResponse(c, requestBodyParsed.FileHandle)
	}

	if err := verifyModifyPermissions(request.PublicKey, requestBodyParsed.FileHandle, file.ModifierHash, c); err != nil {
		return err
	}

	completedFile, err = file.FinishUpload()
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

	if err := account.UseStorageSpaceInByte(int(completedFile.FileSizeInByte)); err != nil {
		utils.DeleteDefaultBucketObjectKeys(completedFile.FileID)
		models.DB.Delete(&completedFile)
		return InternalErrorResponse(c, err)
	}

	if err := utils.SetDefaultObjectCannedAcl(models.GetFileDataKey(completedFile.FileID), utils.CannedAcl_PublicRead); err != nil {
		return InternalErrorResponse(c, err)
	}

	if err := utils.SetDefaultObjectCannedAcl(models.GetFileMetadataKey(completedFile.FileID), utils.CannedAcl_PublicRead); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, fileUploadCompletedRes)
}
