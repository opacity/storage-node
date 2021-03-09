package models

import "time"

// PublicShare ...
type PublicShare struct {
	PublicID   string    `gorm:"primary_key;autoIncrement:false" json:"public_id" binding:"required"`
	CreatedAt  time.Time `json:"createdAt"`
	ViewsCount int       `gorm:"not null" json:"views_count"`
	FileID     string    `gorm:"UNIQUE_INDEX:idx_publicshare;not null" json:"file_id"`
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
