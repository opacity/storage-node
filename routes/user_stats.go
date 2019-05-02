package routes

import (
	"github.com/gin-gonic/gin"
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

func userStats(c *gin.Context) {
	fileSizeInByte := utils.GetMetricCounter(utils.Metrics_FileUploadedSizeInByte_Counter)

	OkResponse(c, userStatsRes{
		UserAccountsCount:    int(utils.GetMetricCounter(utils.Metrics_AccountCreated_Counter)),
		UploadedFilesCount:   int(utils.GetMetricCounter(utils.Metrics_FileUploaded_Counter)),
		UploadedFileSizeInMb: fileSizeInByte / 1000000.0,
	})
}
