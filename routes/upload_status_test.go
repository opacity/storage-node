package routes

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

func testSetupUploadStatus() {
	utils.SetTesting("../.env")
	models.Connect(utils.Env.DatabaseURL)
}

func Test_Init_Upload_Status(t *testing.T) {
	testSetupUploadStatus()
	gin.SetMode(gin.TestMode)
}

func Test_Upload_Status_Bad_Request(t *testing.T) {
}

func Test_Upload_Status_Verification_Failed(t *testing.T) {
}

func Test_Upload_Status_No_Account_Found(t *testing.T) {
}

func Test_Upload_Status_No_File_Found(t *testing.T) {
}

func Test_Upload_Status_File_Not_Finished(t *testing.T) {
}

func Test_Upload_Status_Upload_Finished(t *testing.T) {
}
