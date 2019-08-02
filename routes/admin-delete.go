package routes

import (
	"encoding/hex"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

func AdminDeleteFileHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminDeleteFile)
}

func adminDeleteFile(c *gin.Context) error {
	defer c.Request.Body.Close()

	handle := c.Request.FormValue("fileId")
	if len(handle) < 64 {
		return BadRequestResponse(c, errors.New("bad file ID"))
	}
	fileId := handle[0:64]

	var err error
	var collectedErrors []error

	_, hexErr := hex.DecodeString(fileId)
	if hexErr != nil {
		return BadRequestResponse(c, errors.New("not a valid hex string"))
	}

	var completedFile models.CompletedFile
	if completedFile, err = models.GetCompletedFileByFileID(fileId); err != nil {
		utils.AppendIfError(err, &collectedErrors)
	} else {
		if err := models.DB.Delete(&completedFile).Error; err != nil {
			utils.AppendIfError(err, &collectedErrors)
		}
	}

	if err := utils.DeleteDefaultBucketObjectKeys(fileId); err != nil {
		utils.AppendIfError(err, &collectedErrors)
	}

	if len(collectedErrors) > 0 {
		return InternalErrorResponse(c, utils.CollectErrors(collectedErrors))
	}

	utils.SlackLog("Admin delete successfully called for file handle: " + fileId)

	return OkResponse(c, StatusRes{Status: "delete success"})
}
