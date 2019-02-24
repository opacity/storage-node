package models

import (
	"testing"

	"github.com/opacity/storage-node/utils"
)

func returnValidFile() File {
	return File{
		FileID:              utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
		FileSize:            10,
		FileStorageLocation: "locationOfFile",
		UploadStatus:        FileUploadNotStarted,
		ChunkIndex:          0,
	}
}

func Test_Init_Files(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.DatabaseURL)
}

func Test_Valid_File_Passes(t *testing.T) {
	file := returnValidFile()

	if err := utils.Validator.Struct(file); err != nil {
		t.Fatalf("file should have passed validation but didn't: " + err.Error())
	}
}

//func Test_File_And_Account_Are_Associated(t *testing.T) {
//	account := returnValidAccount()
//	file := returnValidFile()
//
//	// Add account to DB
//	if err := DB.Create(&account).Error; err != nil {
//		t.Fatalf("should have created account but didn't: " + err.Error())
//	}
//
//	// change to associate with the account we created
//	file.AccountID = account.AccountID
//
//	// Add file to DB
//	if err := DB.Create(&file).Error; err != nil {
//		t.Fatalf("should have created file but didn't: " + err.Error())
//	}
//
//	// sanity check to make sure AccountIDs match
//	assert.Equal(t, file.AccountID, account.AccountID)
//
//	// Get the actual rows from the DB since the CreatedAt and UpdatedAt times are truncated
//	actualFile := File{}
//	actualAccount := Account{}
//
//	DB.First(&actualFile, "file_id = ?", file.FileID)
//	DB.First(&actualAccount, "account_id = ?", account.AccountID)
//
//	// Test the associations
//	accounts := []Account{}
//	files := []File{}
//
//	// Test association using Related
//	DB.Model(&actualFile).Related(&accounts)
//	/* This ^ is equivalent to:
//	SELECT * FROM accounts WHERE account_id = file.account_id; */
//	assert.Equal(t, actualAccount, accounts[0])
//
//	// Test association using Association
//	DB.Model(&actualAccount).Association("Files").Find(&files)
//	assert.Equal(t, actualFile, files[0])
//}

func Test_Empty_FileID_Fails(t *testing.T) {
	file := returnValidFile()
	file.FileID = ""

	if err := utils.Validator.Struct(file); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_No_File_Size_Fails(t *testing.T) {
	file := returnValidFile()
	file.FileSize = 0

	if err := utils.Validator.Struct(file); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_No_Storage_Location_Data_Fails(t *testing.T) {
	file := returnValidFile()
	file.FileStorageLocation = ""

	if err := utils.Validator.Struct(file); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_No_UploadStatus_Fails(t *testing.T) {
	file := returnValidFile()
	file.UploadStatus = 0

	if err := utils.Validator.Struct(file); err == nil {
		t.Fatalf("account should have failed validation")
	}
}
