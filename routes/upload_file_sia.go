package routes

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type UploadFileSiaObj struct {
	FileHandle string `form:"fileHandle" validate:"required,len=64" minLength:"64" maxLength:"64" example:"a deterministically created file handle"`
}

type UploadFileSiaReq struct {
	verification
	requestBody
	fileData         string `formFile:"fileData" validate:"required" example:"a binary string of the file data"`
	uploadFileSiaObj UploadFileSiaObj
}

func (v *UploadFileSiaReq) getObjectRef() interface{} {
	return &v.uploadFileSiaObj
}

// UploadFileSiaHandler godoc
// @Summary upload a Sia file
// @Description upload a Sia file via a stream
// @Accept mpfd
// @Produce json
// @Param UploadFileSiaReq body routes.UploadFileSiaReq true "an object to upload a Sia file"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 403 {object} string "signature did not match"
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
	permissionHashKey := getPermissionHashKeyForBadger(models.GetFileMetadataKey(fileID))

	permissionHashInKV, _, err := utils.GetValueFromKV(permissionHashKey)
	if err != nil {
		return ForbiddenResponse(c, errors.New(SignatureDidNotMatchResponse))
	}

	if err := verifyPermissions(request.PublicKey, fileID, permissionHashInKV, c); err != nil {
		return err
	}

	return utils.UploadSiaFile(request.fileData, fileID, false)
}
