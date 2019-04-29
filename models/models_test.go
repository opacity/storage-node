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
	} else {
		DB.Exec("DELETE from accounts;")
	}
}

func deleteFiles(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling deleteAccounts method on test database")
	} else {
		DB.Exec("DELETE from files;")
	}
}

func deleteCompletedFiles(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling deleteAccounts method on test database")
	} else {
		DB.Exec("DELETE from completed_files;")
	}
}
