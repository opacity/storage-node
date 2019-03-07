package routes

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type downloadFileReq struct {
	AccountID string `binding:"required,len=64"`
	UploadID  string `binding:"required"`
}

type downloadFileRes struct {
	// Url should point to S3, thus client does not need to download it from this node.
	FileDownloadUrl string `json:"fileDownloadUrl"`
	// Add other auth-token and expired within a certain period of time.
}

func DownloadFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(downloadFile)
}

func downloadFile(c *gin.Context) {
	request := downloadFileReq{
		AccountID: c.Param("accountID"),
		UploadID:  c.Param("uploadID"),
	}
	if err := utils.Validator.Struct(request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		BadRequestResponse(c, err)
		return
	}

	// validate user
	account, err := models.GetAccountById(request.AccountID)
	if err != nil {
		AccountNotFoundResponse(c, request.AccountID)
		return
	}

	// verify object existed in S3
	objectKey := fmt.Sprintf("%s%s", account.S3Prefix(), request.UploadID)
	if !utils.DoesDefaultBucketObjectExist(objectKey) {
		NotFoundResponse(c, errors.New("Such data does not exist"))
		return
	}

	if err := utils.SetDefaultObjectCannedAcl(objectKey, utils.CannedAcl_PublicRead); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	if err := models.ExpireObject(objectKey); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	url := fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s", utils.Env.AwsRegion, utils.Env.BucketName, objectKey)
	OkResponse(c, downloadFileRes{
		// Redirect to a different URL that client would have authorization to download it.
		FileDownloadUrl: url,
	})
}
