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

func initFileUpload(c *gin.Context) error {
	defer c.Request.Body.Close()

	request := InitFileUploadReq{}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxRequestSize)
	err := c.Request.ParseMultipartForm(MaxRequestSize)

	if err != nil {
		return BadRequestResponse(c, err)
	} else {
		request.PublicKey = c.Request.FormValue("publicKey")
		request.Signature = c.Request.FormValue("signature")
		request.RequestBody = c.Request.FormValue("requestBody")
	}

	multiFile, _, err := c.Request.FormFile("metadata")
	defer multiFile.Close()
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	var fileBytes bytes.Buffer
	_, err = io.Copy(&fileBytes, multiFile)
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	requestBodyParsed := InitFileUploadObj{}

	account, err := returnAccountIfVerifiedFromStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c)
	if err != nil {
		return err
	}

	if err := verifyIfPaid(account, c); err != nil {
		return err
	}

	if err := checkHaveEnoughStorageSpace(account, requestBodyParsed.FileSizeInByte, c); err != nil {
		return err
	}

	objKey, uploadID, err := utils.CreateMultiPartUpload(models.GetFileDataKey(requestBodyParsed.FileHandle))
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	if err := utils.SetDefaultBucketObject(models.GetFileMetadataKey(requestBodyParsed.FileHandle), fileBytes.String()); err != nil {
		return InternalErrorResponse(c, err)
	}

	file := models.File{
		FileID:       requestBodyParsed.FileHandle,
		EndIndex:     requestBodyParsed.EndIndex,
		AwsUploadID:  uploadID,
		AwsObjectKey: objKey,
		ExpiredAt:    account.ExpirationDate(),
	}
	if err := models.DB.Create(&file).Error; err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, InitFileUploadRes{
		Status: "File is init. Please continue to upload",
	})
}

func verifyIfPaid(account models.Account, c *gin.Context) error {
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
		return AccountNotPaidResponse(c, response)
	}
	return nil
}

func checkHaveEnoughStorageSpace(account models.Account, fileSizeInByte int64, c *gin.Context) error {
	inGb := float64(fileSizeInByte) / float64(1e9)
	if inGb+account.StorageUsed > float64(account.StorageLimit) {
		return AccountNotEnoughSpaceResponse(c)
	}
	return nil
}
