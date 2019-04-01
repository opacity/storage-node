package routes

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type uploadFileReq struct {
	AccountID string `form:"accountID" binding:"required,len=64"`
	UploadID  string `form:"uploadID" binding:"required"`
	FileData  string `form:"fileData" binding:"required"`
	FileHash  string `form:"fileHash" binding:"required,len=64"`
	PartIndex int    `form:"partIndex" binding:"required,gte=0"`
	EndIndex  int    `form:"endIndex" binding:"required,gtefield=PartIndex"`
}

type uploadFileRes struct {
	Status string `json:"status"`
}

/*UploadFileHandler is a handler for the user to upload files*/
func UploadFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(uploadFile)
}

func uploadFile(c *gin.Context) {
	request := uploadFileReq{}

	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		BadRequestResponse(c, err)
		return
	}

	account, err := models.GetAccountById(request.AccountID)
	if err != nil {
		AccountNotFoundResponse(c, request.AccountID)
		return
	}

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
		OkResponse(c, response)
		return
	}
	if err != nil {
		InternalErrorResponse(c, err)
		return
	}

	objectKey := fmt.Sprintf("%s%s", account.S3Prefix(), request.UploadID)
	if err := utils.SetDefaultBucketObject(objectKey, request.FileData); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	OkResponse(c, uploadFileRes{
		Status: "File is uploaded",
	})
}
