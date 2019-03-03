package models

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/utils"
)

/*The life cycle of a particular object inside the bucket. */
type S3ObjectLifeCycle struct {
	BucketName  string `gorm:"primary_key"`
	ObjectName  string `gorm:"primary_key"`
	ExpiredTime time.Time
}

func ExpireObject(objectName string) error{
	s := S3ObjectLifeCycle{
		BucketName : utils.Env.BucketName,
		ObjectName: objectName,
		ExpiredTime: time.Now().Add(time.Hour * 1)
	}
	return DB.Create(&s).Error
}
