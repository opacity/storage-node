package models

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func returnValidFile() File {
	return File{
		FileID:           utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
		AwsUploadID:      utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
		AwsObjectKey:     utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
		EndIndex:         10,
		CompletedIndexes: nil,
	}
}

func returnCompletedPart(partNumber int) *s3.CompletedPart {
	return &s3.CompletedPart{
		ETag:       aws.String(utils.RandSeqFromRunes(32, []rune("abcdef01234567890"))),
		PartNumber: aws.Int64(int64(partNumber)),
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
		t.Fatalf("file should have failed validation")
	}
}

func Test_No_AwsUploadID_Fails(t *testing.T) {
	file := returnValidFile()
	file.AwsUploadID = ""

	if err := utils.Validator.Struct(file); err == nil {
		t.Fatalf("file should have failed validation")
	}
}

func Test_No_AwsObjectKey_Fails(t *testing.T) {
	file := returnValidFile()
	file.AwsObjectKey = ""

	if err := utils.Validator.Struct(file); err == nil {
		t.Fatalf("file should have failed validation")
	}
}

func Test_EndIndex_Too_Low_Fails(t *testing.T) {
	file := returnValidFile()
	file.EndIndex = -1

	if err := utils.Validator.Struct(file); err == nil {
		t.Fatalf("file should have failed validation")
	}
}

func Test_UpdateCompletedIndexes(t *testing.T) {
	file := returnValidFile()

	// Add file to DB
	if err := DB.Create(&file).Error; err != nil {
		t.Fatalf("should have created file but didn't: " + err.Error())
	}

	completedPartIndex2 := returnCompletedPart(2)
	completedPartIndex5 := returnCompletedPart(5)

	expectedMap := make(IndexMap)
	expectedMap[*completedPartIndex2.PartNumber] = completedPartIndex2
	expectedMap[*completedPartIndex5.PartNumber] = completedPartIndex5

	err := file.UpdateCompletedIndexes(completedPartIndex2)
	assert.Nil(t, err)
	err = file.UpdateCompletedIndexes(completedPartIndex5)
	assert.Nil(t, err)

	actualFile := File{}
	DB.First(&actualFile, "file_id = ?", file.FileID)
	actualMap := actualFile.GetCompletedIndexesAsMap()

	assert.Equal(t, expectedMap, actualMap)
}

func Test_GetCompletedIndexesAsMap(t *testing.T) {
	file := returnValidFile()

	// Add file to DB
	if err := DB.Create(&file).Error; err != nil {
		t.Fatalf("should have created file but didn't: " + err.Error())
	}

	expectedMap := make(IndexMap)
	startingMap := file.GetCompletedIndexesAsMap()
	assert.Equal(t, expectedMap, startingMap)

	completedPartIndex2 := returnCompletedPart(2)
	completedPartIndex5 := returnCompletedPart(5)

	expectedMap[*completedPartIndex2.PartNumber] = completedPartIndex2
	expectedMap[*completedPartIndex5.PartNumber] = completedPartIndex5

	err := file.UpdateCompletedIndexes(completedPartIndex2)
	assert.Nil(t, err)
	err = file.UpdateCompletedIndexes(completedPartIndex5)
	assert.Nil(t, err)

	actualFile := File{}
	DB.First(&actualFile, "file_id = ?", file.FileID)
	actualMap := actualFile.GetCompletedIndexesAsMap()

	assert.Equal(t, expectedMap, actualMap)
}

func Test_SaveCompletedIndexesAsString(t *testing.T) {
	file := returnValidFile()

	// Add file to DB
	if err := DB.Create(&file).Error; err != nil {
		t.Fatalf("should have created file but didn't: " + err.Error())
	}

	indexMap := make(IndexMap)

	completedPartIndex2 := returnCompletedPart(2)
	completedPartIndex5 := returnCompletedPart(5)

	indexMap[2] = completedPartIndex2
	indexMap[5] = completedPartIndex5

	err := file.SaveCompletedIndexesAsString(indexMap)
	assert.Nil(t, err)

	actualMap := file.GetCompletedIndexesAsMap()

	assert.Equal(t, indexMap, actualMap)
}

func Test_VerifyAllChunksUploaded(t *testing.T) {
	file := returnValidFile()

	// Add file to DB
	if err := DB.Create(&file).Error; err != nil {
		t.Fatalf("should have created file but didn't: " + err.Error())
	}

	indexMap := make(IndexMap)

	completedPartIndex2 := returnCompletedPart(2)
	completedPartIndex5 := returnCompletedPart(5)

	indexMap[2] = completedPartIndex2
	indexMap[5] = completedPartIndex5

	err := file.SaveCompletedIndexesAsString(indexMap)
	assert.Nil(t, err)

	allChunksUploaded := file.VerifyAllChunksUploaded()
	assert.False(t, allChunksUploaded)

	for i := 0; i <= file.EndIndex; i++ {
		indexMap[int64(i)] = returnCompletedPart(i)
	}

	err = file.SaveCompletedIndexesAsString(indexMap)
	assert.Nil(t, err)

	allChunksUploaded = file.VerifyAllChunksUploaded()
	assert.True(t, allChunksUploaded)
}
