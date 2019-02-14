package services

import (
	"encoding/hex"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Eth struct {
	GenerateWallet
}

// GenerateWallet Generate Valid Ethereum Network Address and private key
type GenerateWallet func() (addr common.Address, privateKey string, err error)

var (
	EthWrapper Eth
	MAIN       = "mainnet"
)

func init() {
	EthWrapper = Eth{
		GenerateWallet: generateWallet,
	}
}

// Generate an Ethereum address and private key
func generateWallet() (addr common.Address, privateKey string, err error) {
	ethAccount, err := crypto.GenerateKey()
	if err != nil {
		// TODO:  log this error?
		return addr, "", err
	}
	addr = crypto.PubkeyToAddress(ethAccount.PublicKey)
	privateKey = hex.EncodeToString(ethAccount.D.Bytes())
	if privateKey[0] == '0' || len(privateKey) != 64 || len(addr) != 20 {
		return generateWallet()
	}
	return addr, privateKey, err
}
