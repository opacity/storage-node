package models

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/meirf/gopart"
	"github.com/opacity/storage-node/utils"
)

type FileStorageType int

const (
	S3 FileStorageType = iota + 1
	Sia
	Skynet
)

type CompletedFile struct {
	FileID         string          `gorm:"primary_key" json:"fileID" validate:"required,len=64" minLength:"64" maxLength:"64"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	ExpiredAt      time.Time       `json:"expiredAt"`
	FileSizeInByte int64           `json:"fileSizeInByte" validate:"required"`
	StorageType    FileStorageType `json:"storageType" validate:"required,gte=1" gorm:"default:1"`
	ModifierHash   string          `json:"modifierHash" validate:"required,len=64" minLength:"64" maxLength:"64"`
	ApiVersion     int             `json:"apiVersion" validate:"omitempty,gte=1" gorm:"default:1"`
}

/*BeforeCreate - callback called before the row is created*/
func (completedFile *CompletedFile) BeforeCreate(scope *gorm.Scope) error {
	return utils.Validator.Struct(completedFile)
}

/*BeforeUpdate - callback called before the row is updated*/
func (completedFile *CompletedFile) BeforeUpdate(scope *gorm.Scope) error {
	return utils.Validator.Struct(completedFile)
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
	for fileIDsRange := range gopart.Partition(len(fileIDs), 5000) {
		if err := DB.Where(fileIDs[fileIDsRange.Low:fileIDsRange.High]).Delete(CompletedFile{}).Error; err != nil {
			return err
		}
	}
	return nil
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

/*UpdateExpiredAt receives an array of file handles and updates the ExpiredAt times of any file that matches
one of the file handles*/
func UpdateExpiredAt(fileHandles []string, key string, newExpiredAtTime time.Time) error {
	modifierHashes, err := CreateModifierHashes(fileHandles, key)
	if err != nil {
		return err
	}
	db := DB.Table("completed_files").Where("file_id IN (?) AND modifier_hash IN (?)",
		fileHandles, modifierHashes).Updates(map[string]interface{}{"expired_at": newExpiredAtTime,
		"updated_at": time.Now()})
	if db.Error != nil {
		return db.Error
	}
	if db.RowsAffected != int64(len(fileHandles)) {
		return fmt.Errorf("expected to affect %d rows in UpdateExpiredAt but actually affected %d rows", len(fileHandles), db.RowsAffected)
	}
	return nil
}

/*CreateModifierHashes creates the modifier hashes for the array of file handles passed in*/
func CreateModifierHashes(fileHandles []string, key string) ([]string, error) {
	var modifierHashes []string
	for _, fileHandle := range fileHandles {
		modifierHash, err := utils.HashString(key + fileHandle)
		if err != nil {
			return []string{}, err
		}
		modifierHashes = append(modifierHashes, modifierHash)
	}
	return modifierHashes, nil
}
