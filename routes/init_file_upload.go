package routes

import (
	"fmt"

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
	RequestBody string `json:"requestBody" binding:"required" example:"should produce routes.InitFileUploadObj, see description for example"`
}

type InitFileUploadRes struct {
	Status string `json:"status" example:"Success"`
}

// TODO: Update the godoc
func InitFileUploadHandler() gin.HandlerFunc {
	return ginHandlerFunc(initFileUpload)
}

func initFileUpload(c *gin.Context) {
	request := InitFileUploadReq{}
	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		BadRequestResponse(c, err)
		return
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
