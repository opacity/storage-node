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
	DefaultGasPriceGwei       uint
	SlowGasPriceGwei          uint
	FastGasPriceGwei          uint
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
	sc.WalletPrivateKey, err = services.StringToPrivateKey(utils.DecryptWithGeneratedNonce(utils.Env.EncryptionKey, sc.WalletPrivateKeyEncrypted))
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

func SetWallets() {
	smartContracts := []SmartContract{}
	DB.Find(&smartContracts)

	defaultGasPrice := services.ConvertGweiToWei(big.NewInt(80))

	services.EthOpsWrapper = services.EthOps{
		TransferToken:           services.TransferTokenWrapper,
		TransferETH:             services.TransferETHWrapper,
		GetTokenBalance:         services.GetTokenBalanceWrapper,
		GetETHBalance:           services.GetETHBalanceWrapper,
		CheckForPendingTokenTxs: services.CheckForPendingTokenTxsWrapper,
	}

	services.EthWrappers = make(map[uint]*services.Eth)
	for _, smartContract := range smartContracts {
		// singletons
		services.EthWrappers[smartContract.ID] = &services.Eth{
			AddressNonceMap:                make(map[common.Address]uint64),
			MainWalletAddress:              smartContract.WalletAddress,
			MainWalletPrivateKey:           smartContract.WalletPrivateKey,
			DefaultGasPrice:                services.ConvertGweiToWei(big.NewInt(80)),
			DefaultGasForPaymentCollection: new(big.Int).Mul(defaultGasPrice, big.NewInt(int64(services.GasLimitTokenSend))),
			SlowGasPrice:                   services.ConvertGweiToWei(big.NewInt(80)),
			FastGasPrice:                   services.ConvertGweiToWei(big.NewInt(145)),

			ChainId:         smartContract.NetworkID,
			ContractAddress: smartContract.ContractAddress,
			NodeUrl:         smartContract.NodeURL,
		}
	}
}

// @TODO: remove this after first run with wallets in DB
func MigrateEnvWallets() {
	wallets := []SmartContract{}
	walletsResults := DB.Find(&wallets)

	if walletsResults.RowsAffected == 0 {
		ethMainWallet := SmartContract{
			Network:                   "ethereum",
			NetworkIDuint:             1,
			ContractAddressString:     utils.Env.ContractAddress,
			NodeURL:                   utils.Env.EthNodeURL,
			WalletAddressString:       utils.Env.MainWalletAddress,
			WalletPrivateKeyEncrypted: utils.EncryptWithGeneratedNonce(utils.Env.EncryptionKey, utils.Env.MainWalletPrivateKey),
			DefaultGasPriceGwei:       80,
			SlowGasPriceGwei:          80,
			FastGasPriceGwei:          145,
		}

		if utils.Env.GoEnv != "production" {
			ethMainWallet.Network = "goerli"
			ethMainWallet.NetworkIDuint = 5
		}
		DB.Model(&SmartContract{}).Create(&ethMainWallet)
	}
	SetWallets()
}
