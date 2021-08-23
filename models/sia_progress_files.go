package models

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/utils"
)

// @TODO: Handle Sia file expiration; via cron job? past 60? 30? days after expiration?
type SiaProgressFile struct {
	FileID       string    `gorm:"primary_key" json:"fileID" validate:"required,len=64" minLength:"64" maxLength:"64"`
	CreatedAt    time.Time `json:"createdAt"`
	ExpiredAt    time.Time `json:"expiredAt"`
	ModifierHash string    `json:"modifierHash" validate:"required,len=64" minLength:"64" maxLength:"64"`
	ApiVersion   int       `json:"apiVersion" validate:"omitempty,gte=1" gorm:"default:2"`
}

func (siaProgressFile *SiaProgressFile) BeforeCreate(scope *gorm.Scope) error {
	return utils.Validator.Struct(siaProgressFile)
}

func (siaProgressFile *SiaProgressFile) BeforeUpdate(scope *gorm.Scope) error {
	return utils.Validator.Struct(siaProgressFile)
}

func GetSiaProgressFileById(fileID string) (SiaProgressFile, error) {
	siaProgressFile := SiaProgressFile{}
	err := DB.Where("file_id = ?", fileID).First(&siaProgressFile).Error
	return siaProgressFile, err
}

func DeleteAllExpiredSiaProgressFiles(expiredTime time.Time) error {
	return DB.Where("expired_at < ?", expiredTime).Delete(SiaProgressFile{}).Error
}

func (siaProgressFile *SiaProgressFile) SaveSiaProgressFile() error {
	return DB.Save(&siaProgressFile).Error
}
