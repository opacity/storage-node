package routes

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_User_Stats(t *testing.T) {
	utils.SetTesting("../.env")
	models.Connect(utils.Env.DatabaseURL)
	gin.SetMode(gin.TestMode)
}

func Test_User_Stats(t *testing.T) {
	models.DeleteAccountsForTest(t)
	models.DeleteCompletedFilesForTest(t)

	createNewAccount(t)
	createNewAccount(t)
	createCompletedFile(t, 1000000)
	createCompletedFile(t, 1500000)
	createCompletedFile(t, 2000000)

	w := getUserStats(t)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	assert.Contains(t, w.Body.String(), "\"userAccountsCount\":2,")
	assert.Contains(t, w.Body.String(), "\"uploadedFilesCount\":3,")
	assert.Contains(t, w.Body.String(), "\"uploadedFileSizeInMb\":4.5")
}

func getUserStats(t *testing.T) *httptest.ResponseRecorder {
	router := returnEngine()
	admin := router.Group(AdminPath)
	admin.GET("/user_stats", UserStatsHandler())

	req, err := http.NewRequest(http.MethodGet, admin.BasePath()+"/user_stats", nil)
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
	}
	// Create a response recorder so you can inspect the response
	w := httptest.NewRecorder()
	// Perform the request
	router.ServeHTTP(w, req)

	return w
}

func createCompletedFile(t *testing.T, fileSize int64) models.CompletedFile {
	file := models.CompletedFile{
		FileID:         utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
		FileSizeInByte: fileSize,
	}

	if err := models.DB.Create(&file).Error; err != nil {
		t.Fatalf("should have created CompletedFile but didn't: " + err.Error())
	}
	return file
}

func createNewAccount(t *testing.T) models.Account {
	accountID := utils.RandSeqFromRunes(models.AccountIDLength, []rune("abcdef01234567890"))
	ethAddress, privateKey, _ := services.EthWrapper.GenerateWallet()

	account := models.Account{
		AccountID:            accountID,
		MonthsInSubscription: models.DefaultMonthsPerSubscription,
		StorageLocation:      "https://createdInRoutesUserStatsTest.com/12345",
		StorageLimit:         models.BasicStorageLimit,
		StorageUsed:          10,
		PaymentStatus:        models.InitialPaymentReceived,
		EthAddress:           ethAddress.String(),
		EthPrivateKey:        hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, accountID)),
		MetadataKey:          utils.RandSeqFromRunes(64, []rune("abcdef01234567890")),
	}

	if err := models.DB.Create(&account).Error; err != nil {
		t.Fatalf("should have created account but didn't: " + err.Error())
	}

	return account
}
