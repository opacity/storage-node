package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_PublicShares(t *testing.T) {
	utils.SetTesting("../.env")
	models.Connect(utils.Env.DatabaseURL)
	gin.SetMode(gin.TestMode)
}

func Test_Revoke_PublicShares(t *testing.T) {
	models.DeletePublicSharesForTest(t)
	ps := models.CreateTestPublicShare(t)

	accountID, privateKey := generateValidateAccountId(t)
	CreatePaidAccountForTest(t, accountID)

	publicShareObj := PublicShareObj{
		Shortlink: ps.PublicID,
	}

	v, b := returnValidVerificationAndRequestBody(t, publicShareObj, privateKey)
	publicShareOpsReq := PublicShareOpsReq{
		verification:   v,
		requestBody:    b,
		publicShareObj: publicShareObj,
	}

	w := httpPostRequestHelperForTest(t, "/"+PublicSharePathPrefix+PublicShareRevokePath, "v2", publicShareOpsReq)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Public share revoked")

	t.Cleanup(func() {
		cleanUpBeforeTest(t)
	})
}

func Test_GetUrl_PublicShares(t *testing.T) {
	models.DeletePublicSharesForTest(t)
	ps := createTestPublicShareWithS3Files(t, "")

	requestTestGetPublicShareUrl(t, ps)

	t.Cleanup(func() {
		cleanUpBeforeTest(t)
		ps.RemovePublicShare()
		utils.DeleteDefaultBucketObjectKeys(ps.FileID)
	})
}

func Test_ViewsCountIncreases_PublicShares(t *testing.T) {
	models.DeletePublicSharesForTest(t)
	ps := createTestPublicShareWithS3Files(t, "")
	tries := 6

	for tryNo := 0; tryNo < tries; tryNo++ {
		requestTestGetPublicShareUrl(t, ps)
	}
	psReturned, err := models.GetPublicShareByID(ps.PublicID)
	assert.Nil(t, err)
	assert.Equal(t, tries, psReturned.ViewsCount)

	t.Cleanup(func() {
		cleanUpBeforeTest(t)
		ps.RemovePublicShare()
		utils.DeleteDefaultBucketObjectKeys(ps.FileID)
	})
}

func createTestPublicShareWithS3Files(t *testing.T, fileData string) models.PublicShare {
	ps := models.CreateTestPublicShare(t)
	if fileData == "" {
		fileData = "opacity-public-share-test"
	}
	utils.SetDefaultBucketObject(models.GetFileDataKey(ps.FileID), fileData, "")
	utils.SetDefaultBucketObject(models.GetFileDataPublicKey(ps.FileID), fileData, "")

	return ps
}

func requestTestGetPublicShareUrl(t *testing.T, ps models.PublicShare) *httptest.ResponseRecorder {
	params := map[string]string{
		":shortlink": ps.PublicID,
	}
	w := httpGetRequestHelperForTest(t, "/"+PublicSharePathPrefix+PublicShareShortlinkPath, "v2", params)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), utils.Env.BucketName)

	return w
}
