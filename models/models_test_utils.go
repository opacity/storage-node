package models

import (
	"testing"

	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
	"github.com/teris-io/shortid"
)

func DeleteAccountsForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeleteAccountsForTest method on test database")
	} else {
		DB.Exec("DELETE from accounts;")
	}
}

func DeleteExpiredAccountsForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeleteExpiredAccountsForTest method on test database")
	} else {
		DB.Exec("DELETE from expired_accounts;")
	}
}

func DeleteUpgradesForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeleteUpgradesForTest method on test database")
	} else {
		DB.Exec("DELETE from upgrades;")
	}
}

func DeleteRenewalsForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeleteRenewalsForTest method on test database")
	} else {
		DB.Exec("DELETE from renewals;")
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

func DeleteStripePaymentsForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeleteStripePaymentsForTest method on test database")
	} else {
		DB.Exec("DELETE from stripe_payments;")
	}
}

func DeletePublicSharesForTest(t *testing.T) {
	if utils.Env.DatabaseURL != utils.Env.TestDatabaseURL {
		t.Fatalf("should only be calling DeletePublicSharesForTest method on test database")
	} else {
		DB.Exec("DELETE from public_shares;")
	}
}

func CreateTestPublicShare(t *testing.T) PublicShare {
	ps := CreatePublicShareObj()
	assert.Nil(t, DB.Create(&ps).Error)
	return ps
}

func CreatePublicShareObj() PublicShare {
	shortID, _ := shortid.Generate()
	return PublicShare{
		PublicID:   shortID,
		ViewsCount: 0,
		FileID:     utils.GenerateFileHandle(),
	}
}
