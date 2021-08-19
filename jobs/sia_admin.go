package jobs

import (
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

type siaAdmin struct{}

func AdminSiaStatsHandler(c *gin.Context) {
	c.JSON(200, utils.GetSiaRenter())
}

func (s siaAdmin) Name() string {
	return "siaAdmin"
}

func (s siaAdmin) ScheduleInterval() string {
	return "@every 6h"
}

func (s siaAdmin) Run() {
	utils.SlackLog("running " + s.Name())

}

func (s siaAdmin) Runnable() bool {
	if err := utils.IsSiaClientInit(); err != nil {
		return false
	}
	return true
}
