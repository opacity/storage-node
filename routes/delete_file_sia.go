package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

// DeleteFileSiaHandler godoc
// @Summary delete a Sia file
// @Description delete a Sia file
// @Accept json
// @Produce json
// @Param routes.GenericSiaFileReq body routes.GenericSiaFileReq true "file info object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "the handle of the file",
// @description }
// @Success 200 {object} routes.deleteFileRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v2/sia/delete [post]
/*DeleteFileSiaHandler is a handler for the user to delete a Sia file*/
func DeleteFileSiaHandler() gin.HandlerFunc {
	return ginHandlerFunc(deleteFileSia)
}

func deleteFileSia(c *gin.Context) error {
	if !utils.WritesEnabled() {
		return ServiceUnavailableResponse(c, errMaintenance)
	}

	request := GenericSiaFileReq{}

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

	if err := DeleteFileSiaByID(request.genericFileActionObj.FileHandle, request.PublicKey, account, c); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, deleteFileRes{})
}

func DeleteFileSiaByID(fileID, publicKey string, account models.Account, c *gin.Context) error {
	completedFile, err := models.GetCompletedFileByFileID(fileID)
	if err != nil {
		return err
	}

	if err := verifyAccountPlan(account, models.Sia, c); err != nil {
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

	if err := utils.DeleteSiaFile(fileID); err != nil {
		return err
	}

	if err := models.DB.Delete(&completedFile).Error; err != nil {
		return err
	}

	return nil
}
