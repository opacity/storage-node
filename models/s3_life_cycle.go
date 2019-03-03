package models

import (
	"time"

	"github.com/jinzhu/gorm"
)

/*The life cycle of a particular object inside the bucket. */
type S3ObjectLifeCycle struct {
	BucketName  string `gorm:"primary_key"`
	ObjectName  string `gorm:"primary_key"`
	ExpiredTime time.Time
}
