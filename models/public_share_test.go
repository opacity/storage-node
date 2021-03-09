package models

import (
	"testing"

	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
	"github.com/teris-io/shortid"
)

func Test_Init_PublicShare(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.TestDatabaseURL)
}

func Test_Get_PublicShare_By_ID(t *testing.T) {
	DeletePublicSharesForTest(t)
	s := createTestPublicShare()
	assert.Nil(t, DB.Create(&s).Error)

	publicShare, err := GetPublicShareByID(s.PublicID)
	assert.True(t, publicShare.FileID == s.FileID)
	assert.Nil(t, err)

	t.Cleanup(func() {
		publicShare.RemovePublicShare()
	})
}

func Test_Public_Share_FileID_Unique(t *testing.T) {
	DeletePublicSharesForTest(t)
	s1 := createTestPublicShare()
	s2 := createTestPublicShare()
	assert.Nil(t, DB.Create(&s1).Error)

	s2.FileID = s1.FileID
	err := DB.Create(&s2).Error
	if err == nil {
		t.Fatalf("two shortlinks for the same FileID is not allowed")
	}

	t.Cleanup(func() {
		s1.RemovePublicShare()
	})
}

func Test_Public_Share_Empty_FileID_Fails(t *testing.T) {
	s := createTestPublicShare()
	s.FileID = ""

	if err := utils.Validator.Struct(s); err == nil {
		t.Fatalf("public share with an empty FileID is not allowed")
	}
}

func createTestPublicShare() PublicShare {
	shortID, _ := shortid.Generate()
	return PublicShare{
		PublicID:   shortID,
		ViewsCount: 0,
		FileID:     utils.GenerateFileHandle(),
	}
}
