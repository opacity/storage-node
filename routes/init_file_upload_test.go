package routes

import (
	"crypto/ecdsa"
	"math/big"
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_File_Upload(t *testing.T) {
	setupTests(t)
}

func Test_initFileUploadWithUnpaidAccount(t *testing.T) {
	accountID, privateKey := generateValidateAccountId(t)

	CreateUnpaidAccountForTest(t, accountID)
	req, _ := createValidInitFileUploadRequest(t, 123, 1, privateKey)

	form := map[string]string{
		"metadata": "abc",
	}
	formFile := map[string]string{
		"metadata": "abc_file",
	}
	w := httpPostFormRequestHelperForTest(t, InitUploadPath, &req, form, formFile, "v1")

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `"invoice"`)
	assert.Contains(t, w.Body.String(), `"cost":2`)
}

func Test_initFileUploadWithPaidAccount(t *testing.T) {
	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return true, utils.TestNetworkID, nil
	}
	accountID, privateKey := generateValidateAccountId(t)
	CreatePaidAccountForTest(t, accountID)

	req, uploadObj := createValidInitFileUploadRequest(t, 123, 1, privateKey)

	form := map[string]string{
		"metadata": "abc",
	}
	formFile := map[string]string{
		"metadata": "abc_file",
	}
	w := httpPostFormRequestHelperForTest(t, InitUploadPath, &req, form, formFile, "v1")

	assert.Equal(t, http.StatusOK, w.Code)

	file, err := models.GetFileById(uploadObj.FileHandle)
	assert.Nil(t, err)
	assert.Equal(t, uploadObj.EndIndex, file.EndIndex)
	assert.NotNil(t, file.AwsUploadID)
	assert.NotNil(t, file.AwsObjectKey)
	assert.NotNil(t, file.ModifierHash)

	assert.Nil(t, utils.DeleteDefaultBucketObjectKeys(file.FileID))
}

func Test_initFileUploadWithPaidAccount_MissingFormAndFormFile(t *testing.T) {
	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return true, utils.TestNetworkID, nil
	}
	accountID, privateKey := generateValidateAccountId(t)
	CreatePaidAccountForTest(t, accountID)

	req, _ := createValidInitFileUploadRequest(t, 123, 1, privateKey)

	w := httpPostFormRequestHelperForTest(t, InitUploadPath, &req, nil, nil, "v1")

	assert.Equal(t, http.StatusOK, w.Code)
}

func Test_initFileUploadWithoutEnoughSpace(t *testing.T) {
	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, uint, error) {
		return true, utils.TestNetworkID, nil
	}
	accountID, privateKey := generateValidateAccountId(t)
	account := CreatePaidAccountForTest(t, accountID)

	fileSizeInByte := (int64(account.StorageLimit)-(int64(account.StorageUsedInByte)/1e9))*1e9 + 1
	req, uploadObj := createValidInitFileUploadRequest(t, fileSizeInByte, 1, privateKey)

	form := map[string]string{
		"metadata": "abc",
	}
	formFile := map[string]string{
		"metadata": "abc_file",
	}
	w := httpPostFormRequestHelperForTest(t, InitUploadPath, &req, form, formFile, "v1")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Account does not have enough space")

	_, err := models.GetFileById(uploadObj.FileHandle)
	assert.True(t, gorm.IsRecordNotFoundError(err))
}

func createValidInitFileUploadRequest(t *testing.T, fileSizeInByte int64, endIndex int, privateKey *ecdsa.PrivateKey) (InitFileUploadReq, InitFileUploadObj) {
	uploadObj := InitFileUploadObj{
		FileHandle:     utils.GenerateFileHandle(),
		FileSizeInByte: fileSizeInByte,
		EndIndex:       endIndex,
	}
	v, b := returnValidVerificationAndRequestBody(t, uploadObj, privateKey)
	req := InitFileUploadReq{
		verification: v,
		requestBody:  b,
	}
	return req, uploadObj
}
