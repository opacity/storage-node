package routes

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type UploadFileObj struct {
	ChunkData  string `form:"chunkData" binding:"required" example:"a binary string of the chunk data"`
	FileHandle string `form:"fileHandle" binding:"required,len=64" minLength:"64" maxLength:"64" example:"a deterministically created file handle"`
	PartIndex  int    `form:"partIndex" binding:"required,gte=1" example:"1"`
	EndIndex   int    `form:"endIndex" binding:"required,gtefield=PartIndex" example:"2"`
}

type UploadFileReq struct {
	verification
	UploadFile UploadFileObj `form:"uploadFile" binding:"required"`
}

type uploadFileRes struct {
	Status string `json:"status" example:"Chunk is uploaded"`
}

// UploadFileHandler godoc
// @Summary upload a chunk of a file
// @Description upload a chunk of a file. The first partIndex must be 1. The endIndex must be greater than or equal to partIndex.
// @Accept  mpfd
// @Produce  json
// @Param UploadFileReq body routes.UploadFileReq true "an object to upload a chunk of a file"
// @Success 200 {object} routes.uploadFileRes
// @Failure 403 {object} routes.accountCreateRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/upload [post]
/*UploadFileHandler is a handler for the user to upload files*/
func UploadFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(uploadFile)
}

func uploadFile(c *gin.Context) {
	request := UploadFileReq{}

	if err := utils.ParseRequestBody(c.Request, &request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		BadRequestResponse(c, err)
		return
	}

	account, err := returnAccountIfVerified(request.UploadFile, request.Address, request.Signature, c)
	if err != nil {
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
		AccountNotPaidResponse(c, response)
		return
	}
	if err != nil {
		InternalErrorResponse(c, err)
		return
	}

	file, err := models.GetOrCreateFile(models.File{
		FileID:    request.UploadFile.FileHandle,
		EndIndex:  request.UploadFile.EndIndex,
		ExpiredAt: account.ExpirationDate(),
	})
	if err != nil {
		InternalErrorResponse(c, err)
		return
	}

	var multipartErr error
	var completedPart *s3.CompletedPart
	if request.UploadFile.PartIndex == models.FirstChunkIndex {
		completedPart, multipartErr = handleFirstChunk(file, request.UploadFile.PartIndex, request.UploadFile.ChunkData)
	} else {
		completedPart, multipartErr = handleOtherChunk(file, request.UploadFile.PartIndex, request.UploadFile.ChunkData)
	}

	if multipartErr != nil {
		InternalErrorResponse(c, multipartErr)
		return
	} else {
		file.UpdateCompletedIndexes(completedPart)
	}

	completedFile, err := file.FinishUpload()
	if err != nil && err != models.IncompleteUploadErr {
		utils.LogIfError(err, nil)
	}
	if err == nil {
		err := account.UseStorageSpaceInByte(int(completedFile.FileSizeInByte))
		utils.LogIfError(err, nil)
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
	completedPart, err := utils.UploadMultiPartPart(aws.StringValue(file.AwsObjectKey), aws.StringValue(file.AwsUploadID),
		[]byte(chunkData), chunkIndex)
	return completedPart, err
}
