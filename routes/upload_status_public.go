package routes

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/log"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

// FileUploadCompletedPublicRes ...
type FileUploadCompletedPublicRes struct {
	Shortlink string `json:"shortlink"`
}

type SqsThumbnailMessage struct {
	MimeType string
	S3URL    string
}

type UploadStatusPublicObj struct {
	FileHandle string `json:"fileHandle" binding:"required,len=64" minLength:"64" maxLength:"64" example:"a deterministically created file handle"`
	MimeType   string `form:"mimeType" example:" image/png"`
}

type UploadStatusPublicReq struct {
	verification
	requestBody
	uploadStatusPublicObj UploadStatusPublicObj
}

// CheckUploadStatusPublicHandler godoc
// @Summary check status of a public upload
// @Description check status of a public upload, once it is finished it will trigger a thumbnail generation
// @Accept json
// @Produce json
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
// @description   "mimeType": "image/png"
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 404 {string} string "file not found"
// @Failure 403 {string} string "signature did not match"
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v2/upload-status-public [post]
/*CheckUploadStatusPublicHandler is a handler for checking upload statuses*/
func CheckUploadStatusPublicHandler() gin.HandlerFunc {
	return ginHandlerFunc(checkUploadStatusPublic)
}

func checkUploadStatusPublic(c *gin.Context) error {
	request := UploadStatusPublicReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	fileID := request.uploadStatusPublicObj.FileHandle
	completedFile, completedErr := models.GetCompletedFileByFileID(fileID)
	if completedErr == nil && len(completedFile.FileID) != 0 {
		if utils.DoesDefaultBucketObjectExist(models.GetFileDataPublicKey(fileID)) {
			return OkResponse(c, fileUploadCompletedRes)
		}
	}

	file, err := models.GetFileById(fileID)
	if err != nil || len(file.FileID) == 0 {
		return FileNotFoundResponse(c, fileID)
	}

	if err := verifyPermissions(request.PublicKey, request.uploadStatusPublicObj.FileHandle, file.ModifierHash, c); err != nil {
		return err
	}

	publicShare, err := file.FinishUploadPublic()
	if err != nil {
		if err == models.IncompleteUploadErr {
			incompleteIndexes, err := models.GetIncompleteIndexesAsArray(file.FileID, file.EndIndex)
			if err != nil || len(incompleteIndexes) == 0 {
				// fall back to the old way to get data
				incompleteIndexes = file.GetIncompleteIndexesAsArray()
			}
			return OkResponse(c, missingChunksRes{
				Status:         "chunks missing",
				MissingIndexes: incompleteIndexes,
				EndIndex:       file.EndIndex,
			})
		}
		return InternalErrorResponse(c, err)
	}

	if completedFile, err = models.GetCompletedFileByFileID(completedFile.FileID); err != nil {
		return InternalErrorResponse(c, err)
	}

	fileDataPublicKey := models.GetFileDataPublicKey(completedFile.FileID)
	if err := utils.SetDefaultObjectCannedAcl(fileDataPublicKey, utils.CannedAcl_PublicRead); err != nil {
		return InternalErrorResponse(c, err)
	}

	sqsMessage := SqsThumbnailMessage{
		MimeType: request.uploadStatusPublicObj.MimeType,
		S3URL:    GetS3FileUrl(fileDataPublicKey),
	}
	sqsMessageBody, err := json.Marshal(sqsMessage)
	if err != nil {
		log.Error(err.Error())
	}
	utils.SendSqsMessage(string(sqsMessageBody))

	return OkResponse(c, FileUploadCompletedPublicRes{
		Shortlink: publicShare.PublicID,
	})
}
