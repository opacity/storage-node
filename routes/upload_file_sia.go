package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type UploadFileSiaReq struct {
	verification
	requestBody
	FileData         string `formFile:"fileData" validate:"required" example:"a binary string of the file data"`
	uploadFileSiaObj GenericFileActionObj
}

func (v *UploadFileSiaReq) getObjectRef() interface{} {
	return &v.uploadFileSiaObj
}

// UploadFileSiaHandler godoc
// @Summary upload a Sia file
// @Description upload a Sia file via a form
// @Accept mpfd
// @Produce json
// @Param UploadFileSiaReq body routes.UploadFileSiaReq true "an object to upload a Sia file"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 403 {string} string "signature did not match"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v2/sia/upload [post]
/*UploadFileSiaHandler is a handler for the user to upload a Sia file*/
func UploadFileSiaHandler() gin.HandlerFunc {
	return ginHandlerFunc(uploadFileSia)
}

func uploadFileSia(c *gin.Context) error {
	defer c.Request.Body.Close()

	request := UploadFileSiaReq{}
	if err := verifyAndParseFormRequest(&request, c); err != nil {
		return err
	}

	fileID := request.uploadFileSiaObj.FileHandle

	siaProgressFile, err := models.GetSiaProgressFileById(fileID)
	if err != nil {
		return SiaFileNotInitialised(c)
	}

	if err := verifyPermissions(request.PublicKey, fileID, siaProgressFile.ModifierHash, c); err != nil {
		return err
	}

	// @TODO: Set TTL somehow; cron job?
	// Handle in async
	// if err := utils.UploadSiaFile(request.FileData, fileID, false); err != nil {
	// 	return InternalErrorResponse(c, err)
	// }

	// Fire and forget
	go utils.UploadSiaFile(request.FileData, fileID, false)

	return OkResponse(c, StatusRes{
		Status: "Sia file started uploading",
	})
}
