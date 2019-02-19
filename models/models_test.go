package models

import (
	"testing"

	"github.com/opacity/storage-node/utils"
)

func Test_Init_Models(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.DatabaseURL)
}
