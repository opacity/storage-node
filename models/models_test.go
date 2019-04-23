package models

import (
	"testing"

	"github.com/opacity/storage-node/utils"
)

func Test_Init_Models(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.DatabaseURL)
}

func deleteAccounts(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling deleteAccounts method on test database")
	}

	if err := DB.Unscoped().Delete(&Account{}).Error; err != nil {
		t.Fatalf("should have deleted accounts but didn't: " + err.Error())
	}

	accounts := []Account{}

	DB.Find(&accounts)

	for account := range accounts {
		if err := DB.Unscoped().Delete(&account).Error; err != nil {
			t.Fatalf("should have deleted account but didn't: " + err.Error())
		}
	}
}
