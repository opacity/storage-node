package models

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/utils"
)

// PublicShare ...
type PublicShare struct {
	PublicID   string    `gorm:"primary_key;autoIncrement:false" json:"public_id" validate:"required"`
	CreatedAt  time.Time `json:"createdAt"`
	ViewsCount int       `gorm:"not null" json:"views_count"`
	FileID     string    `gorm:"UNIQUE_INDEX:idx_publicshare;not null" json:"file_id" validate:"required,len=64" minLength:"64" maxLength:"64"`
}

/*BeforeCreate - callback called before the row is created*/
func (publicShare *PublicShare) BeforeCreate(scope *gorm.Scope) error {
	return utils.Validator.Struct(publicShare)
}

/*BeforeUpdate - callback called before the row is updated*/
func (publicShare *PublicShare) BeforeUpdate(scope *gorm.Scope) error {
	return utils.Validator.Struct(publicShare)
}

// GetPublicShareByID returns the public share. If not found, return nil without error.
func GetPublicShareByID(publicID string) (PublicShare, error) {
	publicShare := PublicShare{}
	err := DB.Where("public_id = ?", publicID).First(&publicShare).Error
	return publicShare, err
}

// UpdateViewsCount increments the views count of a PublicShare by 1
func (publicShare *PublicShare) UpdateViewsCount() error {
	return DB.Model(&publicShare).Update("views_count", publicShare.ViewsCount+1).Error
}

// RemovePublicShare removes a public share (revokes it)
func (publicShare *PublicShare) RemovePublicShare() error {
	return DB.Delete(&publicShare).Error
}
