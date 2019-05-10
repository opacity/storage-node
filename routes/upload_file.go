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
	RequestBody string `json:"requestBody" binding:"required" example:"should produce routes.UploadFileObj, see description for example"`
}

type uploadFileRes struct {
	Status string `json:"status" example:"Chunk is uploaded"`
}

var chunkUploadCompletedRes = uploadFileRes{
	Status: "Chunk is uploaded",
}

var fileUploadCompletedRes = uploadFileRes{
	Status: "File is uploaded",
}

// UploadFileHandler godoc
// @Summary upload a chunk of a file
// @Description upload a chunk of a file. The first partIndex must be 1. The endIndex must be greater than or equal to partIndex.
// @Accept  mpfd
// @Produce  json
// @Param UploadFileReq body routes.UploadFileReq true "an object to upload a chunk of a file"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"chunkData": "a binary string of the chunk data",
// @description 	"fileHandle": "a deterministically created file handle",
// @description 	"partIndex": 1,
// @description 	"endIndex": 2
// @description }
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
		BadRequestResponse(c, fmt.Errorf("bad request, unable to parse request body:  %v", err))
		return
	}

	requestBodyParsed := UploadFileObj{}

	account, err := returnAccountIfVerifiedFromStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c)
	if err != nil {
		return
	}

	file, err := models.GetFileById(requestBodyParsed.FileHandle)
	if err != nil || len(file.FileID) == 0 {
		FileNotFoundResponse(c, requestBodyParsed.FileHandle)
		return
	}

	completedPart, multipartErr := handleChunkData(file, requestBodyParsed.PartIndex, requestBodyParsed.ChunkData)
	if multipartErr != nil {
		InternalErrorResponse(c, multipartErr)
		return
	}
	file.UpdateCompletedIndexes(completedPart)

	completedFile, err := file.FinishUpload()
	if err != nil {
		if err == models.IncompleteUploadErr {
			OkResponse(c, chunkUploadCompletedRes)
			return
		}
		InternalErrorResponse(c, err)
		return
	}

	if err := account.UseStorageSpaceInByte(int(completedFile.FileSizeInByte)); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	OkResponse(c, fileUploadCompletedRes)
}

func handleChunkData(file models.File, chunkIndex int, chunkData string) (*s3.CompletedPart, error) {
	return utils.UploadMultiPartPart(aws.StringValue(file.AwsObjectKey), aws.StringValue(file.AwsUploadID),
		[]byte(chunkData), chunkIndex)
}
