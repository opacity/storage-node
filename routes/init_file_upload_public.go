package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

// InitFileUploadPublicHandler godoc
// @Summary start a public upload
// @Description start a public upload.
// @Accept mpfd
// @Produce json
// @Param InitFileUploadReq body routes.InitFileUploadReq true "an object to start a public file upload"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "the file ID for which the public file should be created",
// @description 	"fileSizeInByte": "55600008877",
// @description 	"endIndex": 2
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 403 {string} string "signature did not match / account does not exist"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v2/init-upload-public [post]
/*InitFileUploadPublicHandler is a handler for the user to start uploads*/
func InitFileUploadPublicHandler() gin.HandlerFunc {
	return ginHandlerFunc(initFileUploadPublicWithContext)
}

func initFileUploadPublicWithContext(c *gin.Context) error {
	if !utils.WritesEnabled() {
		return ServiceUnavailableResponse(c, errMaintenance)
	}

	request := InitFileUploadReq{}

	if err := verifyAndParseFormRequest(&request, c); err != nil {
		return err
	}

	return initFileUploadPublicWithRequest(request, c)
}

func initFileUploadPublicWithRequest(request InitFileUploadReq, c *gin.Context) error {
	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	if err := verifyIfPaidWithContext(account, c); err != nil {
		return err
	}

	if err := verifyAccountPlan(account, utils.S3, c); err != nil {
		return err
	}

	objKey, uploadID, err := utils.CreateMultiPartUpload(models.GetFileDataPublicKey(request.initFileUploadObj.FileHandle), "")
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	modifierHash, err := getPermissionHash(request.PublicKey, request.initFileUploadObj.FileHandle, c)
	if err != nil {
		return err
	}

	file := models.File{
		FileID:       request.initFileUploadObj.FileHandle,
		EndIndex:     request.initFileUploadObj.EndIndex,
		AwsUploadID:  uploadID,
		AwsObjectKey: objKey,
		ExpiredAt:    account.ExpirationDate(),
		ModifierHash: modifierHash,
	}

	if err := models.DB.Create(&file).Error; err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, StatusRes{
		Status: "Public file is init. Please continue to upload",
	})
}
