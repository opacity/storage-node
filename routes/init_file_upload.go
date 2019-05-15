package routes

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type InitFileUploadObj struct {
	FileHandle     string `form:"fileHandle" binding:"required,len=64" minLength:"64" maxLength:"64" example:"a deterministically created file handle"`
	FileSizeInByte int64  `form:"fileSizeInByte" binding:"required" example:"200000000000006"`
	EndIndex       int    `form:"endIndex" binding:"required" example:"2"`
}

type InitFileUploadReq struct {
	verification
	requestBody
	Metadata       string `form:"metadata" binding:"required" example:"the metadata of the file you are about to upload, as an array of bytes"`
	MetadataAsFile string `formFile:"metadata"`
	initFileUploadObj InitFileUploadObj
}

type InitFileUploadRes struct {
	Status string `json:"status" example:"Success"`
}

func (v *InitFileUploadReq) getObjectRef() dest interface{} {
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
// @Success 200 {object} routes.InitFileUploadRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 500 {string} string "some information about the internal error"
// @Failure 403 {string} string "signature did not match"
// @Router /api/v1/init-upload [post]
/*InitFileUploadHandler is a handler for the user to start uploads*/
func InitFileUploadHandler() gin.HandlerFunc {
	return ginHandlerFunc(initFileUpload)
}

func initFileUpload(c *gin.Context) {
	request := InitFileUploadReq{}

	if err := verifyAndParseFormRequest(&request, c); err != nil {
		return
	}

	account, err := request.getAccount(c)
	if err != nil {
		return
	}

	if !verifyIfPaid(account, c) {
		return
	}

	if !checkHaveEnoughStorageSpace(account, request.initFileUploadObj.FileSizeInByte, c) {
		return
	}

	objKey, uploadID, err := utils.CreateMultiPartUpload(models.GetFileDataKey(request.initFileUploadObj.FileHandle))
	if err != nil {
		InternalErrorResponse(c, err)
		return
	}

	if err := utils.SetDefaultBucketObject(models.GetFileMetadataKey(request.initFileUploadObj.FileHandle), fileBytes.String()); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	file := models.File{
		FileID:       request.initFileUploadObj.FileHandle,
		EndIndex:     request.initFileUploadObj.EndIndex,
		AwsUploadID:  uploadID,
		AwsObjectKey: objKey,
		ExpiredAt:    account.ExpirationDate(),
	}
	if err := models.DB.Create(&file).Error; err != nil {
		InternalErrorResponse(c, err)
		return
	}

	OkResponse(c, InitFileUploadRes{
		Status: "File is init. Please continue to upload",
	})
}

func verifyIfPaid(account models.Account, c *gin.Context) bool {
	// Check if paid
	paid, err := account.CheckIfPaid()

	if err == nil && !paid {
		cost, _ := account.Cost()
		response := accountCreateRes{
			Invoice: models.Invoice{
				Cost:       cost,
				EthAddress: account.EthAddress,
			},
			ExpirationDate: account.ExpirationDate(),
		}
		AccountNotPaidResponse(c, response)
		return false
	}
	return true
}

func checkHaveEnoughStorageSpace(account models.Account, fileSizeInByte int64, c *gin.Context) bool {
	inGb := float64(fileSizeInByte) / float64(1e9)
	if inGb+account.StorageUsed > float64(account.StorageLimit) {
		AccountNotEnoughSpaceResponse(c)
		return false
	}
	return true
}
