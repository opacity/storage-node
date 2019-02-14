package models

import (
	"testing"

	"github.com/opacity/storage-node/utils"
)

func returnValidFile() File {
	return File{
		FileID:              utils.RandSeqFromRunes(64, []rune("abcdefg01234567890")),
		AccountID:           "",
		FileSize:            10,
		FileStorageLocation: "locationOfFile",
		UploadStatus:        FileUploadNotStarted,
		ChunkIndex:          0,
	}
}

func Test_Init_Files(t *testing.T) {
	utils.SetTesting("../.env")
}

func Test_Valid_File_Passes(t *testing.T) {
	account := returnValidAccount()
	file := returnValidFile()
	file.AccountID = account.AccountID

	if err := Validator.Struct(file); err != nil {
		t.Fatalf("file should have passed validation but didn't: " + err.Error())
	}
}
