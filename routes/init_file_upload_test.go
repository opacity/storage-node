package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

func Test_Init_File_Upload(t *testing.T) {
	utils.SetTesting("../.env")
	models.Connect(utils.Env.DatabaseURL)
	gin.SetMode(gin.TestMode)
}

func Test_initFileUpload(t *testing.T) {
}
