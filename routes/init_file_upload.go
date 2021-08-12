package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type InitFileUploadObj struct {
	FileHandle     string `form:"fileHandle" validate:"required,len=64" minLength:"64" maxLength:"64" example:"a deterministically created file handle"`
	FileSizeInByte int64  `form:"fileSizeInByte" validate:"required" example:"200000000000006"`
	EndIndex       int    `form:"endIndex" validate:"required" example:"2"`
}

type InitFileUploadReq struct {
	verification
	requestBody
	Metadata          string `form:"metadata" validate:"required" example:"the metadata of the file you are about to upload, as an array of bytes"`
	MetadataAsFile    string `formFile:"metadata"`
	initFileUploadObj InitFileUploadObj
}

func (v *InitFileUploadReq) getObjectRef() interface{} {
	return &v.initFileUploadObj
}

// InitFileUploadHandler godoc
// @Summary start an upload
// @Description start an upload
// @Accept  mpfd
// @Produce  json
// @Param InitFileUploadReq body routes.InitFileUploadReq true "an object to start a file upload"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
// @description 	"fileSizeInByte": "200000000000006",
// @description 	"endIndex": 2
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 500 {string} string "some information about the internal error"
// @Failure 403 {string} string "signature did not match"
// @Router /api/v1/init-upload [post]
/*InitFileUploadHandler is a handler for the user to start uploads*/
func InitFileUploadHandler() gin.HandlerFunc {
	return ginHandlerFunc(initFileUploadWithContext)
}

func initFileUploadWithContext(c *gin.Context) error {
	if !utils.WritesEnabled() {
		return ServiceUnavailableResponse(c, maintenanceError)
	}

	request := InitFileUploadReq{}

	if err := verifyAndParseFormRequest(&request, c); err != nil {
		return err
	}

	return initFileUploadWithRequest(request, c)
}

func initFileUploadWithRequest(request InitFileUploadReq, c *gin.Context) error {
	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	if err := verifyIfPaidWithContext(account, c); err != nil {
		return err
	}

	if err := CheckHaveEnoughStorageSpace(account, request.initFileUploadObj.FileSizeInByte, c); err != nil {
		return err
	}

	objKey, uploadID, err := utils.CreateMultiPartUpload(models.GetFileDataKey(request.initFileUploadObj.FileHandle), "")
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	if err := utils.SetDefaultBucketObject(models.GetFileMetadataKey(request.initFileUploadObj.FileHandle), request.MetadataAsFile, ""); err != nil {
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
		Status: "File is init. Please continue to upload",
	})
}

func CheckHaveEnoughStorageSpace(account models.Account, fileSizeInByte int64, c *gin.Context) error {
	plannedInGB := (float64(fileSizeInByte) + float64(account.StorageUsedInByte)) / 1e9
	if plannedInGB > float64(account.StorageLimit) {
		return AccountNotEnoughSpaceResponse(c)
	}
	return nil
}
