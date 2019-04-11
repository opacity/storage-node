package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

type userStatsRes struct {
	UserAccountsCount  int `json:"userAccountsCount"`
	UploadedFilesCount int `json:"uploadedFilesCount"`
}

func UserStatsHandler() gin.HandlerFunc {
	return ginHandlerFunc(userStats)
}

func userStats(c *gin.Context) {
	OkResponse(c, userStatsRes{
		UserAccountsCount:  int(utils.GetMetricCounter(utils.Metrics_AccountCreated_Counter)),
		UploadedFilesCount: int(utils.GetMetricCounter(utils.Metrics_FileUploaded_Counter)),
	})
}
