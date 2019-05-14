package routes

import (
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
	RequestBody string `form:"requestBody" binding:"required" example:"should produce routes.InitFileUploadObj, see description for example"`
	Metadata    string `form:"metadata" binding:"required" example:"the metadata of the file you are about to upload, as an array of bytes"`
}

type InitFileUploadRes struct {
	Status string `json:"status" example:"Success"`
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

	err := c.Request.ParseMultipartForm(utils.MaxMultiPartSize + 10000)
	if err != nil {
		BadRequestResponse(c, err)
		return
	} else {
		request.PublicKey = c.Request.FormValue("publicKey")
		request.Signature = c.Request.FormValue("signature")
		request.Metadata = c.Request.FormValue("metadata")
		request.RequestBody = c.Request.FormValue("requestBody")
	}

	requestBodyParsed := InitFileUploadObj{}

	account, err := returnAccountIfVerifiedFromStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c)
	if err != nil {
		return
	}

	if !verifyIfPaid(account, c) {
		return
	}

	if !checkHaveEnoughStorageSpace(account, requestBodyParsed.FileSizeInByte, c) {
		return
	}

	objKey, uploadID, err := utils.CreateMultiPartUpload(requestBodyParsed.FileHandle)
	if err != nil {
		InternalErrorResponse(c, err)
		return
	}

	if err := utils.SetDefaultBucketObject(models.GetFileMetadataKey(requestBodyParsed.FileHandle), request.Metadata); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	file := models.File{
		FileID:       requestBodyParsed.FileHandle,
		EndIndex:     requestBodyParsed.EndIndex,
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
