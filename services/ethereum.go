package services

import (
	"context"
	"encoding/hex"
	"log"

	"crypto/ecdsa"
	"errors"
	"math/big"

	"fmt"

	"sync"

	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/opacity/storage-node/utils"
)

/*Eth is a struct specifying methods for our ethereum wrapper*/
type Eth struct {
	GenerateWallet
	GetTokenBalance
	GetETHBalance
	TransferToken
	TransferETH
	CheckForPendingTokenTxs
}

/*GenerateWallet - generate valid Ethereum network address and private key*/
type GenerateWallet func() (addr common.Address, privateKey string, err error)

/*GetTokenBalance - check Token balance of an address*/
type GetTokenBalance func(common.Address) /*In Wei Unit*/ *big.Int

/*GetETHBalance - check ETH balance of an address*/
type GetETHBalance func(common.Address) /*In Wei Unit*/ *big.Int

/*TransferToken - send Token from one account to another*/
type TransferToken func(fromAddress common.Address, fromPrivateKey *ecdsa.PrivateKey, toAddr common.Address, opqAmount big.Int) (bool, string, int64)

/*TransferETH - send ETH to an ethereum address*/
type TransferETH func(fromAddress common.Address, fromPrivateKey *ecdsa.PrivateKey, toAddr common.Address, amount *big.Int) (types.Transactions, string, int64, error)

/*CheckForPendingTokenTxs - checks whether a pending token transaction exists*/
type CheckForPendingTokenTxs func(address common.Address) (bool, error)

/*AddressToNonceMap is a type of mapping addresses to nonces*/
type AddressToNonceMap map[common.Address]uint64

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

var (
	/*EthWrapper is an instance of our ethereum wrapper*/
	EthWrapper                     Eth
	client                         *ethclient.Client
	mtx                            sync.Mutex
	chainId                        = params.MainnetChainConfig.ChainID
	addressNonceMap                AddressToNonceMap
	MainWalletAddress              common.Address
	MainWalletPrivateKey           *ecdsa.PrivateKey
	DefaultGasPrice                = utils.ConvertGweiToWei(big.NewInt(2))
	DefaultGasForPaymentCollection = new(big.Int).Mul(DefaultGasPrice, big.NewInt(int64(GasLimitTokenSend)))
)

func init() {
	EthWrapper = Eth{
		GenerateWallet:          generateWallet,
		TransferToken:           transferToken,
		TransferETH:             transferETH,
		GetTokenBalance:         getTokenBalance,
		GetETHBalance:           getETHBalance,
		CheckForPendingTokenTxs: checkForPendingTokenTxs,
	}

	addressNonceMap = make(AddressToNonceMap)
}

/*SetWallet gets the address and private key for storage node's main wallet*/
func SetWallet() error {
	var err error
	if utils.Env.MainWalletPrivateKey == "" || utils.Env.MainWalletAddress == "" {
		err = errors.New("need MainWalletAddress and MainWalletPrivateKey for storage node's main wallet")
		utils.LogIfError(err, nil)
		utils.PanicOnError(err)
	}
	MainWalletAddress = common.HexToAddress(utils.Env.MainWalletAddress)
	MainWalletPrivateKey, err = StringToPrivateKey(utils.Env.MainWalletPrivateKey)
	utils.LogIfError(err, nil)
	return err
}

// Shared client provides access to the underlying Ethereum client
func sharedClient() (c *ethclient.Client, err error) {
	if client != nil {
		return client, nil
	}
	// check-lock-check pattern to avoid excessive locking.
	mtx.Lock()
	defer mtx.Unlock()

	if client != nil {
		return client, nil
	}

	c, err = ethclient.Dial(utils.Env.EthNodeURL)
	if err != nil {
		utils.LogIfError(err, nil)
		return
	}
	// Sets Singleton
	client = c
	return
}

// Generate an Ethereum address and private key
func generateWallet() (addr common.Address, privateKey string, err error) {
	ethAccount, err := crypto.GenerateKey()
	if err != nil {
		utils.LogIfError(err, nil)
		return addr, "", err
	}
	addr = crypto.PubkeyToAddress(ethAccount.PublicKey)
	privateKey = hex.EncodeToString(ethAccount.D.Bytes())
	if privateKey[0] == '0' || len(privateKey) != 64 || len(addr) != 20 {
		return generateWallet()
	}
	return addr, privateKey, err
}

// Check balance from a valid address
func getTokenBalance(address common.Address) *big.Int {
	// connect ethereum client
	client, err := sharedClient()
	if err != nil {
		utils.LogIfError(err, nil)
		return big.NewInt(-1)
	}

	// instance of the token contract
	OpacityAddress := common.HexToAddress(utils.Env.ContractAddress)
	Opacity, err := NewOpacity(OpacityAddress, client)
	if err != nil {
		utils.LogIfError(err, nil)
		return big.NewInt(-1)
	}
	callOpts := bind.CallOpts{Pending: false, From: OpacityAddress}
	balance, err := Opacity.BalanceOf(&callOpts, address)
	if err != nil {
		utils.LogIfError(err, nil)
		return big.NewInt(-1)
	}
	return balance
}

// Check balance from a valid ethereum network address
func getETHBalance(addr common.Address) *big.Int {
	// connect ethereum client
	client, err := sharedClient()
	if err != nil {
		utils.LogIfError(err, nil)
	}

	balance, err := client.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		utils.LogIfError(err, nil)
		return big.NewInt(-1)
	}
	return balance
}

func transferToken(from common.Address, privateKey *ecdsa.PrivateKey, to common.Address, opqAmount big.Int) (bool, string, int64) {
	msg := TokenCallMsg{
		From:       from,
		To:         to,
		Amount:     opqAmount,
		Gas:        GasLimitTokenSend,
		PrivateKey: *privateKey,
		TotalWei:   *big.NewInt(0).SetUint64(uint64(opqAmount.Int64())),
	}

	client, _ := sharedClient()
	Opacity, err := NewOpacity(common.HexToAddress(utils.Env.ContractAddress), client)

	if err != nil {
		utils.LogIfError(err, nil)
	}

	// initialize transactor // may need to move this to a session based transactor
	auth := bind.NewKeyedTransactor(&msg.PrivateKey)
	if err != nil {
		utils.LogIfError(err, nil)
	}

	log.Printf("authorized transactor : %v\n", auth.From.Hex())

	// use this when in production:
	gasPrice, err := getGasPrice()

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
		utils.LogIfError(errors.New(fmt.Sprintf("transfer failed: %v", err.Error())), nil)
		return false, "", int64(-1)
	}

	log.Printf("transfer pending: 0x%x\n", tx.Hash())

	printTx(tx)

	return true, tx.Hash().Hex(), int64(tx.Nonce())
}

// Transfer funds from main wallet
func transferETH(fromAddress common.Address, fromPrivKey *ecdsa.PrivateKey, toAddr common.Address, amount *big.Int) (types.Transactions, string, int64, error) {

	client, err := sharedClient()
	if err != nil {
		return types.Transactions{}, "", -1, err
	}

	// initialize the context
	ctx, cancel := createContext()
	defer cancel()

	// generate nonce
	nonce, _ := client.PendingNonceAt(ctx, fromAddress)

	if lastNonce, inMap := ReturnLastNonceFromMap(fromAddress); inMap && nonce <= lastNonce {
		nonce = lastNonce + 1
	}

	UpdateLastNonceInMap(fromAddress, nonce)

	gasPrice, _ := getGasPrice()

	balance := getETHBalance(fromAddress)
	fmt.Printf("balance : %v\n", balance)

	// amount is greater than balance, return error
	if amount.Uint64() > balance.Uint64() {
		return types.Transactions{}, "", -1, fmt.Errorf("balance too low to proceed, send ETH to: %v",
			fromAddress.Hex())
	}

	// create new transaction
	tx := types.NewTransaction(nonce, toAddr, amount, GasLimitETHSend, gasPrice, nil)

	// signer
	signer := types.NewEIP155Signer(chainId)

	// sign transaction
	signedTx, err := types.SignTx(tx, signer, fromPrivKey)
	if err != nil {
		utils.LogIfError(err, nil)
		return types.Transactions{}, "", -1, err
	}

	// send transaction
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		utils.LogIfError(fmt.Errorf("error sending transaction from %s to %s.  Error:  %v",
			fromAddress.String(), signedTx.To().String(), err), nil)
		return types.Transactions{}, "", -1, err
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

func checkForPendingTokenTxs(address common.Address) (bool, error) {
	// connect ethereum client
	client, err := sharedClient()
	if err != nil {
		utils.LogIfError(err, nil)
		return false, err
	}

	// instance of the token contract
	OpacityAddress := common.HexToAddress(utils.Env.ContractAddress)
	Opacity, err := NewOpacity(OpacityAddress, client)
	if err != nil {
		utils.LogIfError(err, nil)
		return false, err
	}
	callOpts := bind.CallOpts{Pending: true, From: OpacityAddress}
	balance, err := Opacity.BalanceOf(&callOpts, address)
	if err != nil {
		utils.LogIfError(err, nil)
		return false, err
	}
	return balance.Int64() > big.NewInt(0).Int64(), nil
}

/*RemoveFromAddressNonceMap removes a key with a certain address from the map of addresses to their
most recent nonces.  This is to prevent us from accidentally using a nonce that is already in the process
of being used, for a particular address.*/
func RemoveFromAddressNonceMap(address common.Address) {
	if _, ok := addressNonceMap[address]; ok && address != MainWalletAddress {
		delete(addressNonceMap, address)
	}
}

/*ReturnLastNonceFromMap returns the latest nonce from the addressNonceMap for a particular
address.*/
func ReturnLastNonceFromMap(address common.Address) (uint64, bool) {
	if _, ok := addressNonceMap[address]; ok {
		return addressNonceMap[address], true
	}
	return uint64(0), false
}

/*UpdateLastNonceInMap updates the last nonce in the addressNonceMap for an address*/
func UpdateLastNonceInMap(address common.Address, lastNonce uint64) {
	addressNonceMap[address] = lastNonce
}

// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
// execution for new transaction
func getGasPrice() (*big.Int, error) {
	// if QAing, un-comment out the line immediately below to hard-code a high gwei value for fast txs
	return utils.ConvertGweiToWei(big.NewInt(2)), nil

	// connect ethereum client
	client, err := sharedClient()
	if err != nil {
		utils.LogIfError(err, nil)
	}

	// there is no guarantee with estimate gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		utils.LogIfError(err, nil)
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
	fmt.Printf("tx to     : %v\n", tx.To().Hash().String())
	fmt.Printf("tx hash   : %v\n", tx.Hash().String())
	fmt.Printf("tx amount : %v\n", tx.Value())
	fmt.Printf("tx cost   : %v\n", tx.Cost())
}

/*StringToAddress converts a string to a common.Address*/
func StringToAddress(address string) common.Address {
	return common.HexToAddress(address)
}

/* StringToPrivateKey Utility HexToECDSA parses a secp256k1 private key*/
func StringToPrivateKey(hexPrivateKey string) (*ecdsa.PrivateKey, error) {
	return crypto.HexToECDSA(hexPrivateKey)
}
