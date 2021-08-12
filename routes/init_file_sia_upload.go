package routes

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type InitFileSiaUploadObj struct {
	FileHandle     string `form:"fileHandle" validate:"required,len=64" minLength:"64" maxLength:"64" example:"a deterministically created file handle"`
	FileSizeInByte int64  `form:"fileSizeInByte" validate:"required" example:"200000000000006"`
}

type InitFileSiaUploadReq struct {
	verification
	requestBody
	MetadataAsFile       string `formFile:"metadata" validate:"required"`
	initFileSiaUploadObj InitFileSiaUploadObj
}

func (v *InitFileSiaUploadReq) getObjectRef() interface{} {
	return &v.initFileSiaUploadObj
}

// InitFileSiaUploadHandler godoc
// @Summary start a Sia file upload
// @Description start a Sia file upload
// @Accept mpfd
// @Produce json
// @Param InitFileSiaUploadReq body routes.InitFileSiaUploadReq true "an object to start a Sia file upload"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
// @description 	"fileSizeInByte": "200000000000006",
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 500 {string} string "some information about the internal error"
// @Failure 403 {string} string "signature did not match"
// @Router /api/v2/sia/init-upload [post]
/*InitFileSiaUploadHandler is a handler for the user to start the upload of a Sia file*/
func InitFileSiaUploadHandler() gin.HandlerFunc {
	return ginHandlerFunc(initFileSiaUploadWithContext)
}

func initFileSiaUploadWithContext(c *gin.Context) error {
	if !utils.WritesEnabled() {
		return ServiceUnavailableResponse(c, maintenanceError)
	}

	request := InitFileSiaUploadReq{}

	if err := verifyAndParseFormRequest(&request, c); err != nil {
		return err
	}

	return initFileSiaUploadWithRequest(request, c)
}

func initFileSiaUploadWithRequest(request InitFileSiaUploadReq, c *gin.Context) error {
	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	if err := verifyIfPaidWithContext(account, c); err != nil {
		return err
	}

	if err := CheckHaveEnoughStorageSpace(account, request.initFileSiaUploadObj.FileSizeInByte, c); err != nil {
		return err
	}

	// @TODO: This might not be needed, but store it for the moment
	ttl := time.Until(account.ExpirationDate())
	if err := utils.BatchSet(&utils.KVPairs{
		models.GetFileMetadataKey(request.initFileSiaUploadObj.FileHandle): request.MetadataAsFile,
	}, ttl); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, StatusRes{
		Status: "Sia file initialised, please proceed with the actual upload",
	})
}
