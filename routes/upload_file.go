package routes

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type uploadFileReq struct {
	AccountID string `form:"accountID" binding:"required,len=64"`
	ChunkHash string `form:"chunkHash" binding:"required"`
	ChunkData string `form:"chunkData" binding:"required"`
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

	file, err := models.GetOrCreateFile(models.File{
		FileID:   request.FileHash,
		EndIndex: request.EndIndex,
	})
	if err != nil {
		InternalErrorResponse(c, err)
		return
	}

	var multipartErr error
	var completedPart *s3.CompletedPart
	if file.AwsUploadID == nil && file.AwsObjectKey == nil {
		completedPart, multipartErr = handleFirstChunk(file, request.PartIndex, request.ChunkData)
	} else {
		completedPart, multipartErr = handleOtherChunk(file, request.PartIndex, request.ChunkData)
	}

	if multipartErr != nil {
		InternalErrorResponse(c, multipartErr)
		return
	} else {
		file.UpdateCompletedIndexes(completedPart)
	}

	OkResponse(c, uploadFileRes{
		Status: "Chunk is uploaded",
	})
}

func handleFirstChunk(file *models.File, chunkIndex int, chunkData string) (*s3.CompletedPart, error) {
	key, uploadID, err := utils.CreateMultiPartUpload(file.FileID)
	if err != nil {
		return nil, err
	}
	err = file.UpdateKeyAndUploadID(key, uploadID)
	if err != nil {
		return nil, err
	}

	return handleOtherChunk(file, chunkIndex, chunkData)
}

func handleOtherChunk(file *models.File, chunkIndex int, chunkData string) (*s3.CompletedPart, error) {
	return utils.UploadMultiPartPart(aws.StringValue(file.AwsObjectKey), aws.StringValue(file.AwsUploadID),
		[]byte(chunkData), chunkIndex)
}
