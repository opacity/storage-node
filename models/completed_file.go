package models

import (
	"fmt"
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
	var value int64
	err := DB.Model(&CompletedFile{}).Select("sum(file_size_in_byte)").Scan(&value).Error
	fmt.Printlf("value message %v", value)

	rows, err := db.Table("completed_file").Select("sum(file_size_in_byte) AS total").Rows()
	if err != nil {
		fmt.Printf("Error : %v", err)
		//return 0, err
	}
	defer rows.Close()

	if rows.Next() {
		total := int64(0)
		err := rows.Scan(&total)
		if err != nil {
			fmt.Printf("Error : %v", err)
			//return 0, err
		}
		fmt.Printf("totatl value: %v", total)
		//return total, nil
	}
	return value, err
}
