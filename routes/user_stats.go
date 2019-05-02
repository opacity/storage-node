package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type userStatsRes struct {
	UserAccountsCount    int     `json:"userAccountsCount"`
	UploadedFilesCount   int     `json:"uploadedFilesCount"`
	UploadedFileSizeInMb float64 `json:"uploadedFileSizeInMb"`
}

func UserStatsHandler() gin.HandlerFunc {
	return ginHandlerFunc(userStats)
}

func userStats(c *gin.Context) {
	userCount := 0
	if err := models.DB.Model(&models.Account{}).Count(&userCount).Error; err != nil {
		utils.LogIfError(err, nil)
		InternalErrorResponse(c, err)
		return
	}

	fileCount := 0
	if err := models.DB.Model(&models.CompletedFile{}).Count(&fileCount).Error; err != nil {
		utils.LogIfError(err, nil)
		InternalErrorResponse(c, err)
		return
	}

	fileSizeInByte, err := models.GetTotalFileSizeInByte()
	if err != nil {
		utils.LogIfError(err, nil)
		InternalErrorResponse(c, err)
		return
	}

	OkResponse(c, userStatsRes{
		UserAccountsCount:    userCount,
		UploadedFilesCount:   fileCount,
		UploadedFileSizeInMb: float64(fileSizeInByte) / 1000000.0,
	})
}
