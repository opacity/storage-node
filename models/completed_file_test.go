package models

import (
	"testing"

	"github.com/opacity/storage-node/utils"
)

func Test_Init_Completed_File(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.TestDatabaseURL)
}
