package models

import (
	"time"
)

/*The life cycle of a particular object inside the bucket. */
type S3ObjectLifeCycle struct {
	ObjectName  string `gorm:"primary_key"`
	ExpiredTime time.Time
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
