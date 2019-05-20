package models

import (
	"testing"

	"github.com/opacity/storage-node/utils"
)

func DeleteAccountsForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeleteAccountsForTest method on test database")
	} else {
		DB.Exec("DELETE from accounts;")
	}
}

func DeleteFilesForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeleteFilesForTest method on test database")
	} else {
		DB.Exec("DELETE from files;")
	}
}

func DeleteCompletedFilesForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeleteCompletedFilesForTest method on test database")
	} else {
		DB.Exec("DELETE from completed_files;")
	}
}

func DeleteCompletedUploadIndexesForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeleteCompletedUploadIndexesForTest method on test database")
	} else {
		DB.Exec("DELETE from completed_upload_indices;")
	}
}
