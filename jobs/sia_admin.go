package jobs

import (
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

func AdminSiaStatsHandler(c *gin.Context) {
	c.JSON(200, utils.GetSiaRenter())
}
