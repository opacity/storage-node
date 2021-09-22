package models

import (
	"crypto/ecdsa"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

/* Defines the smart contracts address and related data */
type SmartContract struct {
	ID                        uint `gorm:"primary_key" `
	Network                   string
	NetworkIDuint             uint   `gorm:"column:network_id"`
	ContractAddressString     string `gorm:"column:contract_address"`
	NodeURL                   string
	WalletAddressString       string `gorm:"column:wallet_address"`
	WalletPrivateKeyEncrypted string `gorm:"column:wallet_private_key"`
	CreatedAt                 time.Time
	UpdatedAt                 time.Time

	NetworkID        *big.Int          `gorm:"-"`
	WalletAddress    common.Address    `gorm:"-"`
	WalletPrivateKey *ecdsa.PrivateKey `gorm:"-"`
	ContractAddress  common.Address    `gorm:"-"`
}

func (sc *SmartContract) AfterFind(tx *gorm.DB) (err error) {
	sc.NetworkID = big.NewInt(int64(sc.NetworkIDuint))
	sc.WalletAddress = services.StringToAddress(sc.WalletAddressString)
	sc.WalletPrivateKey, err = services.StringToPrivateKey(utils.DecryptWithoutNonce(utils.Env.EncryptionKey, sc.WalletPrivateKeyEncrypted))
	sc.ContractAddress = services.StringToAddress(sc.ContractAddressString)
	utils.LogIfError(err, nil)
	return
}

func GetAllSmartContracts() ([]SmartContract, error) {
	sc := []SmartContract{}
	scResults := DB.Find(&sc)

	if scResults.RowsAffected == 0 {
		return sc, errors.New("no smart contracts are configured")
	}

	return sc, nil
}
