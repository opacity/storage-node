package models

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/utils"
)

type CompletedUploadIndex struct {
	FileID string `gorm:"primary_key" json:"fileID" validate:"required"`
	Index  int    `gorm:"primary_key;auto_increment:false" json:"index" validate:"required"`
	Etag   string `json:"etag" validate:"required"`
}

/*BeforeCreate - callback called before the row is created*/
func (completedUploadIndex *CompletedUploadIndex) BeforeCreate(scope *gorm.Scope) error {
	return utils.Validator.Struct(completedUploadIndex)
}

/*BeforeUpdate - callback called before the row is updated*/
func (completedUploadIndex *CompletedUploadIndex) BeforeUpdate(scope *gorm.Scope) error {
	return utils.Validator.Struct(completedUploadIndex)
}

func CreateCompletedUploadIndex(fileID string, index int, etag string) error {
	c := CompletedUploadIndex{
		FileID: fileID,
		Index:  index,
		Etag:   etag,
	}
	return DB.Create(&c).Error
}

func DeleteCompletedUploadIndexes(fileID string) error {
	return DB.Where("file_id = ?", fileID).Delete(&CompletedUploadIndex{}).Error
}

func GetCompletedUploadProgress(fileID string) (int, error) {
	var count int
	err := DB.Model(&CompletedUploadIndex{}).Where("file_id = ?", fileID).Count(&count).Error
	return count, err
}

func GetCompletedPartsAsArray(fileID string) ([]*s3.CompletedPart, error) {
	var completedParts []*s3.CompletedPart
	completedIndexes := []CompletedUploadIndex{}
	if err := DB.Where("file_id = ?", fileID).Order("index").Find(&completedIndexes).Error; err != nil {
		return completedParts, err
	}

	for _, index := range completedIndexes {
		completedParts = append(completedParts, &s3.CompletedPart{
			ETag:       aws.String(index.Etag),
			PartNumber: aws.Int64(int64(index.Index)),
		})
	}
	return completedParts, nil
}

func GetIncompleteIndexesAsArray(fileID string, endIndex int) ([]int64, error) {
	var incompletedIndex []int64
	completedIndexes := []CompletedUploadIndex{}
	if err := DB.Where("file_id = ?", fileID).Order("index").Find(&completedIndexes).Error; err != nil {
		return incompletedIndex, err
	}

	m := make(map[int]bool)
	for _, idx := range completedIndexes {
		m[idx.Index] = true
	}
	for index := FirstChunkIndex; index <= endIndex; index++ {
		if _, ok := m[index]; !ok {
			incompletedIndex = append(incompletedIndex, int64(index))
		}
	}
	return incompletedIndex, nil
}
