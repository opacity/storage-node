package services

import (
	"context"
	"encoding/hex"
	"log"

	"crypto/ecdsa"
	"math/big"

	"fmt"

	"sync"

	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/hashicorp/go-retryablehttp"
)

// needed as singleton in order to change it during tests
type EthOps struct {
	GetTokenBalance
	GetETHBalance
	TransferToken
	TransferETH
	CheckForPendingTokenTxs
}

type Eth struct {
	client                         *ethclient.Client
	mtx                            sync.Mutex
	AddressNonceMap                map[common.Address]uint64
	MainWalletAddress              common.Address
	MainWalletPrivateKey           *ecdsa.PrivateKey
	DefaultGasPrice                *big.Int
	DefaultGasForPaymentCollection *big.Int
	SlowGasPrice                   *big.Int
	FastGasPrice                   *big.Int
	ChainId                        *big.Int
	ContractAddress                common.Address
	NodeUrl                        string
}

/*GetTokenBalance - check Token balance of an address*/
type GetTokenBalance func(ethWrapper *Eth, address common.Address) /*In Wei Unit*/ *big.Int

/*GetETHBalance - check ETH balance of an address*/
type GetETHBalance func(ethWrapper *Eth, address common.Address) /*In Wei Unit*/ *big.Int

/*TransferToken - send Token from one account to another*/
type TransferToken func(ethWrapper *Eth, fromAddress common.Address, fromPrivateKey *ecdsa.PrivateKey, toAddr common.Address, opctAmount big.Int, gasPrice *big.Int) (bool, string, int64)

/*TransferETH - send ETH to an ethereum address*/
type TransferETH func(ethWrapper *Eth, fromAddress common.Address, fromPrivateKey *ecdsa.PrivateKey, toAddr common.Address, amount *big.Int) (types.Transactions, string, int64, error)

/*CheckForPendingTokenTxs - checks whether a pending token transaction exists*/
type CheckForPendingTokenTxs func(*Eth, common.Address) bool

var EthWrappers map[uint]*Eth
var EthOpsWrapper EthOps

/*TokenCallMsg is the message to send to the ETH blockchain to do token transactions*/
type TokenCallMsg struct {
	From       common.Address
	To         common.Address
	Amount     big.Int
	PrivateKey ecdsa.PrivateKey
	Gas        uint64
	GasPrice   big.Int
	TotalWei   big.Int
	Data       []byte
}

// Limits selected based on actual transactions from etherscan
const (
	// Token Transaction Gas Limit
	GasLimitTokenSend uint64 = 54501
	// ETH Gas Limit
	GasLimitETHSend uint64 = 21000
)

// Shared client provides access to the underlying Ethereum client
func (eth *Eth) SharedClient() (c *ethclient.Client) {
	if eth.client != nil {
		return eth.client
	}
	// check-lock-check pattern to avoid excessive locking.
	eth.mtx.Lock()
	defer eth.mtx.Unlock()

	if eth.client != nil {
		return eth.client
	}

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 5
	rpcClient, err := rpc.DialHTTPWithClient(eth.NodeUrl, retryClient.StandardClient())
	if err != nil {
		panic(err)
	}
	c = ethclient.NewClient(rpcClient)

	// Sets Singleton
	eth.client = c
	return
}

// Generate an Ethereum address and private key
func GenerateWallet() (addr common.Address, privateKey string) {
	ethAccount, err := crypto.GenerateKey()
	if err != nil {
		ethereumErrorLog(err, nil)
		return
	}
	addr = crypto.PubkeyToAddress(ethAccount.PublicKey)
	privateKey = hex.EncodeToString(ethAccount.D.Bytes())
	if privateKey[0] == '0' || len(privateKey) != 64 || len(addr) != 20 {
		return GenerateWallet()
	}
	return addr, privateKey
}

// Check balance from a valid address
func GetTokenBalanceWrapper(ethWrapper *Eth, address common.Address) *big.Int {
	// connect ethereum client
	client := ethWrapper.SharedClient()

	// instance of the token contract
	Opacity, err := NewOpacity(ethWrapper.ContractAddress, client)
	if err != nil {
		ethereumErrorLog(err, nil)
		return big.NewInt(-1)
	}
	callOpts := bind.CallOpts{Pending: false, From: ethWrapper.ContractAddress}
	balance, err := Opacity.BalanceOf(&callOpts, address)
	if err != nil {
		ethereumErrorLog(err, nil)
		return big.NewInt(-1)
	}
	return balance
}

// Check balance from a valid ethereum network address
func GetETHBalanceWrapper(ethWrapper *Eth, addr common.Address) *big.Int {
	// connect ethereum client
	client := ethWrapper.SharedClient()

	balance, err := client.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		panic(err)
	}
	return balance
}

func TransferTokenWrapper(ethWrapper *Eth, from common.Address, privateKey *ecdsa.PrivateKey, to common.Address, opctAmount big.Int, gasPrice *big.Int) (bool, string, int64) {
	msg := TokenCallMsg{
		From:       from,
		To:         to,
		Amount:     opctAmount,
		Gas:        GasLimitTokenSend,
		PrivateKey: *privateKey,
	}

	client := ethWrapper.SharedClient()
	Opacity, err := NewOpacity(ethWrapper.ContractAddress, client)
	if err != nil {
		ethereumErrorLog(err, nil)
	}

	// @TODO: initialize transactor // may need to move this to a session based transactor
	auth, err := bind.NewKeyedTransactorWithChainID(&msg.PrivateKey, ethWrapper.ChainId)
	if err != nil {
		ethereumErrorLog(err, nil)
	}

	log.Printf("authorized transactor : %v\n", auth.From.Hex())

	opts := bind.TransactOpts{
		From:     auth.From,
		Signer:   auth.Signer,
		GasLimit: GasLimitTokenSend,
		GasPrice: gasPrice,
		Nonce:    auth.Nonce,
		Context:  auth.Context,
	}

	tx, err := Opacity.Transfer(&opts, msg.To, &msg.Amount)
	if err != nil {
		ethereumErrorLog(err, nil)
		return false, "", int64(-1)
	}

	log.Printf("transfer pending: 0x%x\n", tx.Hash())

	printTx(tx)

	return true, tx.Hash().Hex(), int64(tx.Nonce())
}

// Transfer funds from main wallet
func TransferETHWrapper(ethWrapper *Eth, fromAddress common.Address, fromPrivKey *ecdsa.PrivateKey, toAddr common.Address, amount *big.Int) (types.Transactions, string, int64, error) {
	client := ethWrapper.SharedClient()

	// initialize the context
	ctx, cancel := createContext()
	defer cancel()

	// generate nonce
	nonce, _ := client.PendingNonceAt(ctx, fromAddress)
	if lastNonce, inMap := ethWrapper.ReturnLastNonceFromMap(fromAddress); inMap && nonce <= lastNonce {
		nonce = lastNonce + 1
	}

	ethWrapper.UpdateLastNonceInMap(fromAddress, nonce)

	gasPrice, _ := ethWrapper.GetGasPrice()

	balance := GetETHBalanceWrapper(ethWrapper, fromAddress)
	fmt.Printf("balance : %v\n", balance)

	// amount is greater than balance, return error
	if amount.Uint64() > balance.Uint64() {
		return types.Transactions{}, "", -1, fmt.Errorf("balance too low to proceed, send ETH to: %v",
			fromAddress.Hex())
	}

	// create new transaction
	tx := types.NewTransaction(nonce, toAddr, amount, GasLimitETHSend, gasPrice, nil)

	// signer
	signer := types.NewEIP155Signer(ethWrapper.ChainId)

	// sign transaction
	signedTx, err := types.SignTx(tx, signer, fromPrivKey)
	if err != nil {
		ethereumErrorLog(err, nil)
		return types.Transactions{}, "", -1, err
	}

	// send transaction
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return types.Transactions{}, "", -1, fmt.Errorf("error sending transaction from %s to %s.  Error:  %v",
			fromAddress.String(), signedTx.To().String(), err)
	}

	// pull signed transaction(s)
	signedTxs := types.Transactions{signedTx}
	for tx := range signedTxs {
		transaction := signedTxs[tx]
		printTx(transaction)
	}

	// return signed transactions
	return signedTxs, signedTx.Hash().Hex(), int64(signedTx.Nonce()), nil
}

func CheckForPendingTokenTxsWrapper(ethWrapper *Eth, address common.Address) bool {
	client := ethWrapper.SharedClient()

	// instance of the token contract
	Opacity, err := NewOpacity(ethWrapper.ContractAddress, client)
	if err != nil {
		ethereumErrorLog(err, nil)
		return false
	}
	callOpts := bind.CallOpts{Pending: true, From: ethWrapper.ContractAddress}
	balance, err := Opacity.BalanceOf(&callOpts, address)
	if err != nil {
		ethereumErrorLog(err, nil)
		return false
	}
	return balance.Cmp(big.NewInt(0)) > 0
}

/*RemoveFromAddressNonceMap removes a key with a certain address from the map of addresses to their
most recent nonces.  This is to prevent us from accidentally using a nonce that is already in the process
of being used, for a particular address.*/
func (eth *Eth) RemoveFromAddressNonceMap(address common.Address) {
	if _, ok := eth.AddressNonceMap[address]; ok && address != eth.MainWalletAddress {
		delete(eth.AddressNonceMap, address)
	}
}

/*ReturnLastNonceFromMap returns the latest nonce from the addressNonceMap for a particular
address.*/
func (eth *Eth) ReturnLastNonceFromMap(address common.Address) (uint64, bool) {
	if _, ok := eth.AddressNonceMap[address]; ok {
		return eth.AddressNonceMap[address], true
	}
	return uint64(0), false
}

/*UpdateLastNonceInMap updates the last nonce in the addressNonceMap for an address*/
func (eth *Eth) UpdateLastNonceInMap(address common.Address, lastNonce uint64) {
	(*eth).AddressNonceMap[address] = lastNonce
}

// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
// execution for new transaction
func (eth *Eth) GetGasPrice() (*big.Int, error) {
	// if QAing, un-comment out the line immediately below to hard-code a high gwei value for fast txs
	return (*eth).DefaultGasPrice, nil

	client := eth.SharedClient()

	// there is no guarantee with estimate gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		panic(err)
	}
	return gasPrice, nil
}

// utility context helper to include the deadline initialization
func createContext() (ctx context.Context, cancel context.CancelFunc) {
	deadline := time.Now().Add(5000 * time.Millisecond)
	return context.WithDeadline(context.Background(), deadline)
}

// utility to print
func printTx(tx *types.Transaction) {
	fmt.Printf("tx to        : %v\n", tx.To().Hash().String())
	fmt.Printf("tx hash      : %v\n", tx.Hash().String())
	fmt.Printf("tx amount    : %v\n", tx.Value())
	fmt.Printf("tx cost      : %v\n", tx.Cost())
	fmt.Printf("tx chainId   : %v\n", tx.ChainId().String())
}

/*StringToAddress converts a string to a common.Address*/
func StringToAddress(address string) common.Address {
	return common.HexToAddress(address)
}

/* StringToPrivateKey Utility HexToECDSA parses a secp256k1 private key*/
func StringToPrivateKey(hexPrivateKey string) (*ecdsa.PrivateKey, error) {
	return crypto.HexToECDSA(hexPrivateKey)
}

func ethereumErrorLog(err error, extraInfo map[string]interface{}) {
	if err == nil {
		return
	}
	fmt.Println(err)
	fmt.Println(extraInfo)
}
