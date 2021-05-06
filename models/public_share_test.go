package models

import (
	"testing"

	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Get_PublicShare_By_ID(t *testing.T) {
	ps, err := CreateTestPublicShare()
	assert.Nil(t, err)
	publicShare, err := GetPublicShareByID(ps.PublicID)
	assert.True(t, publicShare.FileID == ps.FileID)
	assert.Nil(t, err)
}
func Test_Public_Share_Empty_FileID_Fails(t *testing.T) {
	ps := CreatePublicShareObj()
	ps.FileID = ""

	if err := utils.Validator.Struct(ps); err == nil {
		t.Fatalf("public share with an empty FileID is not allowed")
	}
}
