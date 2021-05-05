package jobs

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

func TestMain(m *testing.M) {
	utils.SetTesting("../.env")
	models.Connect(utils.Env.TestDatabaseURL)
	gin.SetMode(gin.TestMode)
}
