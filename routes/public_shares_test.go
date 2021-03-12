package routes

import (
	"net/http"
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
