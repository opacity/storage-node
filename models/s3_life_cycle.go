package models

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/utils"
)

/*The life cycle of a particular object inside the bucket. */
type S3ObjectLifeCycle struct {
	ObjectName  string `gorm:"primary_key"`
	ExpiredTime time.Time
}

/*BeforeCreate - callback called before the row is created*/
func (s3ObjectLifeCycle *S3ObjectLifeCycle) BeforeCreate(scope *gorm.Scope) error {
	return utils.Validator.Struct(s3ObjectLifeCycle)
}

/*BeforeUpdate - callback called before the row is updated*/
func (s3ObjectLifeCycle *S3ObjectLifeCycle) BeforeUpdate(scope *gorm.Scope) error {
	return utils.Validator.Struct(s3ObjectLifeCycle)
}

func ExpireObject(objectName string) error {
	expiredTime := time.Now().Add(time.Hour * 1)
	var s = S3ObjectLifeCycle{}

	if DB.Where("object_name = ?", objectName).First(&s).RecordNotFound() {
		s = S3ObjectLifeCycle{
			ObjectName:  objectName,
			ExpiredTime: expiredTime,
		}
		return DB.Create(&s).Error
	}

	s.ExpiredTime = expiredTime
	return DB.Save(&s).Error
}
