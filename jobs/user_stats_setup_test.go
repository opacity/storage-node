package jobs

import (
	"testing"
)

func Test_Init_User_Stats_Setup(t *testing.T) {
	utils.SetTesting("../.env")
	models.Connect(utils.Env.DatabaseURL)
}
