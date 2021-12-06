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
	GenericFileActionObj
	PartIndex int `form:"partIndex" validate:"required,gte=1" example:"1"`
}

type UploadFileReq struct {
	verification
	requestBody
	ChunkData     string `formFile:"chunkData" validate:"required" example:"a binary string of the chunk data"`
	uploadFileObj UploadFileObj
}

var chunkUploadCompletedRes = StatusRes{
	Status: "Chunk is uploaded",
}

var fileUploadCompletedRes = StatusRes{
	Status: "File is uploaded",
}

func (v *UploadFileReq) getObjectRef() interface{} {
	return &v.uploadFileObj
}

// UploadFileHandler godoc
// @Summary upload a chunk of a file
// @Description upload a chunk of a file. The first partIndex must be 1.
// @Accept  mpfd
// @Produce  json
// @Param UploadFileReq body routes.UploadFileReq true "an object to upload a chunk of a file"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
// @description 	"partIndex": 1,
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 403 {object} routes.accountCreateRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/upload [post]
/*UploadFileHandler is a handler for the user to upload files*/
func UploadFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(uploadFile)
}

func uploadFile(c *gin.Context) error {
	defer c.Request.Body.Close()

	request := UploadFileReq{}

	if err := verifyAndParseFormRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	storageType := account.PlanInfo.FileStorageType

	return uploadChunk(request, c, storageType)
}

func uploadChunk(request UploadFileReq, c *gin.Context, storageType utils.FileStorageType) error {
	fileID := request.uploadFileObj.FileHandle
	file, err := models.GetFileById(fileID)
	if err != nil || len(file.FileID) == 0 {
		return FileNotFoundResponse(c, fileID)
	}

	if err := verifyPermissions(request.PublicKey, fileID,
		file.ModifierHash, c); err != nil {
		return err
	}

	fileSize := len(request.ChunkData)
	isLastChunk := request.uploadFileObj.PartIndex == file.EndIndex
	if !isLastChunk && fileSize < int(utils.MinMultiPartSize) {
		return BadRequestResponse(c, fmt.Errorf("upload chunk is %v and does not meet min fileSize %v", fileSize, utils.MinMultiPartSize))
	}

	completedPart, multipartErr := handleChunkData(file, request.uploadFileObj.PartIndex, []byte(request.ChunkData), storageType)
	if multipartErr != nil {
		return InternalErrorResponse(c, multipartErr)
	}

	err = models.CreateCompletedUploadIndex(file.FileID, int(*completedPart.PartNumber), *completedPart.ETag)
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, chunkUploadCompletedRes)
}

func handleChunkData(file models.File, chunkIndex int, chunkData []byte, storageType utils.FileStorageType) (*s3.CompletedPart, error) {
	return utils.UploadMultiPartPart(aws.StringValue(file.AwsObjectKey), aws.StringValue(file.AwsUploadID),
		chunkData, chunkIndex, storageType)
}
