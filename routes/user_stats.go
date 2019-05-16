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

// UserStatsHandler godoc
// @Summary get statistics
// @Description get statistics
// @Accept  json
// @Produce  json
// @Success 200 {object} routes.userStatsRes
// @Router /admin/user_stats [get]
/*UserStatsHandler returns user statistics*/
func UserStatsHandler() gin.HandlerFunc {
	return ginHandlerFunc(userStats)
}

func userStats(c *gin.Context) error {
	userCount := 0
	if err := models.DB.Model(&models.Account{}).Count(&userCount).Error; err != nil {
		return InternalErrorResponse(c, err)
	}

	fileCount := 0
	if err := models.DB.Model(&models.CompletedFile{}).Count(&fileCount).Error; err != nil {
		return InternalErrorResponse(c, err)
	}

	fileSizeInByte, err := models.GetTotalFileSizeInByte()
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, userStatsRes{
		UserAccountsCount:    userCount,
		UploadedFilesCount:   fileCount,
		UploadedFileSizeInMb: float64(fileSizeInByte) / 1000000.0,
	})
}
