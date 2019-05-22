package routes

import (
	"net/http"

	"bytes"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type UploadFileObj struct {
	FileHandle string `form:"fileHandle" binding:"required,len=64" minLength:"64" maxLength:"64" example:"a deterministically created file handle"`
	PartIndex  int    `form:"partIndex" binding:"required,gte=1" example:"1"`
}

type UploadFileReq struct {
	verification
	ChunkData   string `form:"chunkData" binding:"required" example:"a binary string of the chunk data"`
	RequestBody string `form:"requestBody" binding:"required" example:"should produce routes.UploadFileObj, see description for example"`
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
// @Description upload a chunk of a file. The first partIndex must be 1.
// @Accept  mpfd
// @Produce  json
// @Param UploadFileReq body routes.UploadFileReq true "an object to upload a chunk of a file"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
// @description 	"partIndex": 1,
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

func uploadFile(c *gin.Context) error {
	defer c.Request.Body.Close()

	request := UploadFileReq{}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxRequestSize)
	err := c.Request.ParseMultipartForm(MaxRequestSize)
	if err != nil {
		return BadRequestResponse(c, err)
	} else {
		request.PublicKey = c.Request.FormValue("publicKey")
		request.Signature = c.Request.FormValue("signature")
		request.RequestBody = c.Request.FormValue("requestBody")
	}

	multiFile, _, err := c.Request.FormFile("chunkData")
	defer multiFile.Close()
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	var fileBytes bytes.Buffer
	_, err = io.Copy(&fileBytes, multiFile)
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	requestBodyParsed := UploadFileObj{}

	if err := verifyAndParseStringRequest(request.RequestBody, &requestBodyParsed, request.verification, c); err != nil {
		return BadRequestResponse(c, err)
	}

	file, err := models.GetFileById(requestBodyParsed.FileHandle)
	if err != nil || len(file.FileID) == 0 {
		return FileNotFoundResponse(c, requestBodyParsed.FileHandle)
	}

	if err := verifyModifyPermissions(request.PublicKey, requestBodyParsed.FileHandle, file.ModifierHash, c); err != nil {
		return err
	}

	completedPart, multipartErr := handleChunkData(file, requestBodyParsed.PartIndex, fileBytes.Bytes())
	if multipartErr != nil {
		return InternalErrorResponse(c, multipartErr)
	}
	err = models.CreateCompletedUploadIndex(file.FileID, int(*completedPart.PartNumber), *completedPart.ETag)
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, chunkUploadCompletedRes)
}

func handleChunkData(file models.File, chunkIndex int, chunkData []byte) (*s3.CompletedPart, error) {
	return utils.UploadMultiPartPart(aws.StringValue(file.AwsObjectKey), aws.StringValue(file.AwsUploadID),
		chunkData, chunkIndex)
}
