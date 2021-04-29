package models

import (
	"testing"

	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_PublicShare(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.TestDatabaseURL)
}

func Test_Get_PublicShare_By_ID(t *testing.T) {
	DeletePublicSharesForTest()
	ps, err := CreateTestPublicShare()
	assert.Nil(t, err)
	publicShare, err := GetPublicShareByID(ps.PublicID)
	assert.True(t, publicShare.FileID == ps.FileID)
	assert.Nil(t, err)

	t.Cleanup(func() {
		publicShare.RemovePublicShare()
	})
}
func Test_Public_Share_Empty_FileID_Fails(t *testing.T) {
	DeletePublicSharesForTest()
	ps := CreatePublicShareObj()
	ps.FileID = ""

	if err := utils.Validator.Struct(ps); err == nil {
		t.Fatalf("public share with an empty FileID is not allowed")
	}

	t.Cleanup(func() {
		ps.RemovePublicShare()
	})
}
