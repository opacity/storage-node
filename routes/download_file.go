package routes

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type downloadFileObj struct {
	FileID string `json:"fileID" binding:"required" example:"the handle of the file"`
}

type downloadFileReq struct {
	verification
	RequestBody string `json:"requestBody" binding:"required" example:"should produce routes.downloadFileObj, see description for example"`
}

type downloadFileRes struct {
	// Url should point to S3, thus client does not need to download it from this node.
	FileDownloadUrl string `json:"fileDownloadUrl" example:"a URL to use to download the file"`
	// Add other auth-token and expired within a certain period of time.
}

// DownloadFileHandler godoc
// @Summary download a file
// @Description download a file
// @Accept  json
// @Produce  json
// @Param downloadFileReq body routes.downloadFileReq true "download object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileID": "the handle of the file",
// @description }
// @Success 200 {object} routes.downloadFileRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 404 {string} string "such data does not exist"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/download [get]
/*DownloadFileHandler handles the downloading of a file*/
func DownloadFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(downloadFile)
}

func downloadFile(c *gin.Context) {
	request := downloadFileReq{}

	if err := utils.Validator.Struct(request); err != nil {
		err = fmt.Errorf("bad request, unable to parse request body:  %v", err)
		BadRequestResponse(c, err)
		return
	}

	requestBodyParsed := downloadFileObj{}

	var err error
	if _, err = returnAccountIfVerified_v2(request.RequestBody, &requestBodyParsed, request.Address,
		request.Signature, c); err != nil {
		return
	}

	// verify object existed in S3
	if !utils.DoesDefaultBucketObjectExist(requestBodyParsed.FileID) {
		NotFoundResponse(c, errors.New("such data does not exist"))
		return
	}

	if err := utils.SetDefaultObjectCannedAcl(requestBodyParsed.FileID, utils.CannedAcl_PublicRead); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	if err := models.ExpireObject(requestBodyParsed.FileID); err != nil {
		InternalErrorResponse(c, err)
		return
	}

	url := fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s", utils.Env.AwsRegion, utils.Env.BucketName,
		requestBodyParsed.FileID)
	OkResponse(c, downloadFileRes{
		// Redirect to a different URL that client would have authorization to download it.
		FileDownloadUrl: url,
	})
}
