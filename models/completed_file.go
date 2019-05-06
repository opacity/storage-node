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
	rows, err := DB.Model(&CompletedFile{}).Select("sum(file_size_in_byte) AS total").Rows()
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	total := int64(0)
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
	}
	return total, nil
}

/*GetCompletedFileByFileID - return completed file object(first one) if there is not any error. */
func GetCompletedFileByFileID(fileID string) (CompletedFile, error) {
	completedFile := CompletedFile{}
	err := DB.Where("file_id = ?", fileID).First(&completedFile).Error
	return completedFile, err
}
