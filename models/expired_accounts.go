package models

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

/*Account defines a model for managing a user subscription for uploads*/
type ExpiredAccount struct {
	AccountID  string    `gorm:"primary_key" json:"accountID" binding:"required,len=64"` // some hash of the user's master handle
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	EthAddress string    `json:"ethAddress" binding:"required,len=42" minLength:"42" maxLength:"42" example:"a 42-char eth address with 0x prefix"` // the eth address they will send payment to
	ExpiredAt  time.Time `json:"expiredAt"`
	DeletedAt  time.Time `json:"deletedAt"`
}

/*BeforeCreate - callback called before the row is created*/
func (expiredAccount *ExpiredAccount) BeforeCreate(scope *gorm.Scope) error {
	return utils.Validator.Struct(expiredAccount)
}

/*BeforeUpdate - callback called before the row is updated*/
func (expiredAccount *ExpiredAccount) BeforeUpdate(scope *gorm.Scope) error {
	return utils.Validator.Struct(expiredAccount)
}

/*BeforeDelete - callback called before the row is deleted*/
func (expiredAccount *ExpiredAccount) BeforeDelete(scope *gorm.Scope) error {
	return nil
}
