package models

type CompletedUploadIndex struct {
	FileID string `gorm:"primary_key" json:"fileID" binding:"required"`
	Index  int    `gorm:"primary_key;auto_increment:false" json:"index" binding:"required"`
}

func CreateCompletedUploadIndex(fileID string, index int) error {
	c := CompletedUploadIndex{
		FileID: fileID,
		Index:  index,
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
