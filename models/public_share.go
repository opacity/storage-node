package models

import (
	"errors"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/meirf/gopart"
	"github.com/opacity/storage-node/utils"
	"github.com/teris-io/shortid"
)

// PublicShare ...
type PublicShare struct {
	PublicID        string                `gorm:"UNIQUE_INDEX:idx_publicshare;primary_key;autoIncrement:false" json:"public_id" validate:"required"`
	CreatedAt       time.Time             `json:"createdAt"`
	ViewsCount      int                   `gorm:"not null" json:"views_count"`
	Title           string                `gorm:"not null;size:65535" json:"title"`
	Description     string                `gorm:"not null;size:65535" json:"description"`
	MimeType        string                `gorm:"not null;size:255" json:"mimeType"`
	FileExtension   string                `gorm:"not null;size:255" json:"fileExtension"`
	FileID          string                `gorm:"not null" json:"file_id" validate:"required,len=64" minLength:"64" maxLength:"64"`
	FileStorageType utils.FileStorageType `gorm:"not null;default:1" json:"storage_type" validate:"required,len=1"`
}

// CreateShortlinkObj...
type CreateShortlinkObj struct {
	FileID        string `json:"file_id" binding:"required,len=64" minLength:"64" maxLength:"64" example:"the id of the file"`
	Title         string `json:"title" binding:"required" minLength:"1" maxLength:"65535" example:"LoremIpsum"`
	MimeType      string `json:"mimeType" minLength:"1" maxLength:"255" example:"image/png"`
	FileExtension string `json:"fileExtension" minLength:"1" maxLength:"255" example:"png"`
	Description   string `json:"description" binding:"required" minLength:"1" maxLength:"65535" example:"lorem ipsum"`
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

func RemovePublicSharesByIds(fileIDs []string) error {
	for fileIDsRange := range gopart.Partition(len(fileIDs), 5000) {
		if err := DB.Where("file_id IN (?)", fileIDs[fileIDsRange.Low:fileIDsRange.High]).Delete(PublicShare{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func RemovePublicSharesById(fileID string) error {
	return DB.Where("file_id = ?", fileID).Delete(PublicShare{}).Error
}

func CreatePublicShare(createShortlinkObj CreateShortlinkObj) (PublicShare, error) {
	shortID, err := shortid.Generate()
	if err != nil {
		return PublicShare{}, err
	}
	completedFile, err := GetCompletedFileByFileID(createShortlinkObj.FileID)
	if err != nil {
		return PublicShare{}, err
	}
	publicShare := PublicShare{
		PublicID:      shortID,
		ViewsCount:    0,
		Title:         createShortlinkObj.Title,
		Description:   createShortlinkObj.Description,
		MimeType:      createShortlinkObj.MimeType,
		FileExtension: createShortlinkObj.FileExtension,
		FileID:        completedFile.FileID,
	}
	if createShortlinkObj.MimeType == "" {
		publicShare.MimeType = "image/png"
	}
	if createShortlinkObj.FileExtension == "" {
		publicShare.FileExtension = "png"
	}

	if err := DB.Save(&publicShare).Error; err != nil {
		return PublicShare{}, errors.New("error saving the public share")
	}

	return publicShare, nil
}

func GetPublicFileDownloadData(fileID string, storageType utils.FileStorageType) (fileURL, thumbnailURL string) {
	storageURL := utils.GetStorageURL(storageType)

	fileURL = storageURL + GetFileDataPublicKey(fileID)
	thumbnailKey := GetPublicThumbnailKey(fileID)
	thumbnailURL = "https://s3.us-east-2.amazonaws.com/opacity-public/thumbnail_default.png"
	if utils.DoesDefaultBucketObjectExist(thumbnailKey, storageType) {
		thumbnailURL = storageURL + thumbnailKey
	}

	return
}
