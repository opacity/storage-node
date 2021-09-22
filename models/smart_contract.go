package models

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/utils"
)

/* Defines the smart contracts address and related data */
type SmartContract struct {
	ID               uint `gorm:"primary_key" `
	Network          string
	NetworkID        uint
	Address          string
	NodeURL          string
	WalletAddress    string
	WalletPrivateKey string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (sc *SmartContract) AfterFind(tx *gorm.DB) (err error) {
	privateKeyDencrypted := utils.DecryptWithoutNonce(utils.Env.EncryptionKey, sc.WalletPrivateKey)
	sc.WalletPrivateKey = privateKeyDencrypted
	return
}
