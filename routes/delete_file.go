package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type deleteFileObj struct {
	FileID string `json:"fileID" validate:"required" example:"the handle of the file"`
}

type deleteFileReq struct {
	verification
	requestBody
	deleteFileObj deleteFileObj
}

type deleteFileRes struct {
}

func (v *deleteFileReq) getObjectRef() interface{} {
	return &v.deleteFileObj
}

// DeleteFileHandler godoc
// @Summary delete a file
// @Description delete a file
// @Accept  json
// @Produce  json
// @Param deleteFileReq body routes.deleteFileReq true "file deletion object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileID": "the handle of the file",
// @description }
// @Success 200 {object} routes.deleteFileRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/delete [post]
/*DeleteFileHandler is a handler for the user to upload files*/
func DeleteFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(deleteFile)
}

func deleteFile(c *gin.Context) error {
	if !utils.WritesEnabled() {
		return ServiceUnavailableResponse(c, errMaintenance)
	}

	request := deleteFileReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	if err := verifyIfPaidWithContext(account, c); err != nil {
		return err
	}

	if err := DeleteFileByID(request.deleteFileObj.FileID, request.PublicKey, account, c); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, deleteFileRes{})
}

func DeleteFileByID(fileID, publicKey string, account models.Account, c *gin.Context) error {
	completedFile, err := models.GetCompletedFileByFileID(fileID)
	if err != nil {
		return err
	}

	if err := verifyPermissions(publicKey, fileID, completedFile.ModifierHash, c); err != nil {
		return err
	}

	err = account.UseStorageSpaceInByte(int64(-1) * completedFile.FileSizeInByte)
	if err != nil && err.Error() == models.StorageUsedTooLow {
		err = models.DB.Model(&account).Update("storage_used_in_byte", int64(0)).Error
	}
	if err != nil && err.Error() != models.StorageUsedTooLow {
		return err
	}

	if err := utils.DeleteDefaultBucketObjectKeys(fileID); err != nil {
		return err
	}

	if err := models.DB.Delete(&completedFile).Error; err != nil {
		return err
	}

	if err := models.RemovePublicSharesById(fileID); err != nil {
		return err
	}

	return nil
}
