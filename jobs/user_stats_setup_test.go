package jobs

import (
	"testing"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

func Test_Init_User_Stats_Setup(t *testing.T) {
	utils.SetTesting("../.env")
	models.Connect(utils.Env.DatabaseURL)
}
