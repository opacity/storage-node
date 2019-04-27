package models

import (
	"time"

	"github.com/opacity/storage-node/utils"
)

type CompletedFile struct {
	FileID         string    `gorm:"primary_key" json:"fileID" binding:"required"`
	CreatedAt      time.Time `json:"createdAt"`
	ExpiredAt      time.Time `json:"expiredAt"`
	FileSizeInByte int64     `json:"fileSizeInByte"`
}

type sumResult struct {
	total int64
}

func GetAllExpiredCompletedFiles(expiredTime time.Time) ([]string, error) {
	files := []CompletedFile{}
	if err := DB.Where("expired_at < ?", expiredTime).Find(&files).Error; err != nil {
		utils.LogIfError(err, nil)
		return nil, err
	}
	var fileIDs []string
	for _, f := range files {
		fileIDs = append(fileIDs, f.FileID)
	}
	return fileIDs, nil
}

func DeleteAllCompletedFiles(fileIDs []string) error {
	return DB.Where(fileIDs).Delete(CompletedFile{}).Error
}

func GetTotalFileSizeInByte() (int64, error) {
	var value []int
	err := DB.Model(&CompletedFile{}).Select("SUM(file_size_in_byte)").Scan(&value).Error
	return value[0], err
}
