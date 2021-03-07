package models

import "time"

// PublicShare ...
type PublicShare struct {
	PublicID   string    `gorm:"primary_key" json:"public_id" binding:"required"`
	CreatedAt  time.Time `json:"createdAt"`
	ViewsCount int       `json:"views_count"`
	FileID     string    `gorm:"foreignKey:FileID"`
}

// GetPublicShareByID returns the public share. If not found, return nil without error.
func GetPublicShareByID(publicID string) (PublicShare, error) {
	publicShare := PublicShare{}
	err := DB.Where("public_id = ?", publicID).First(&publicShare).Error
	return publicShare, err
}
