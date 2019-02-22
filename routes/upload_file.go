package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type uploadFileReq struct {
	AccountID string `json:"accountID" binding:"required,len=64"`
	UploadID  string `json:"uploadID" binding:"required"`
	FileData  string `json:"fileData" binding:"required"`
}

func UploadFileHandler() gin.HandlerFunc {
	return gin.HandlerFunc(uploadFile)
}

func uploadFile(c *gin.Context) {
	c.JSON(http.StatusOK, "stub for uploading a file with an existing subscription")
}
