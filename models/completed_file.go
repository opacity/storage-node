package models

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/utils"
)

type CompletedFile struct {
	FileID    string    `gorm:"primary_key" json:"FileID" binding:"required"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiredAt time.Time `json:"expiredAt"`
}

func GetAllExpiredFiles(expiredTime time.Time) ([]string, error) {
	files := []CompletedFile{}
	if err := DB.Where("expired_at > ?", expiredTime).Find(&files).Error; err != nil {
		utils.LogIfError(err, nil)
		return nil, err
	}
	var fileIDs []string
	for f := range files {
		fileIDs = append(fileIDs, f.FileID)
	}
	return f, nil
}
