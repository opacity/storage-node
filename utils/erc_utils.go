package utils

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

const (
	/*TokenInWeiUnit is the multiplier for token denominations.*/
	TokenInWeiUnit = 1e18
)

/*ConvertToWeiUnit converts token unit to wei unit. */
func ConvertToWeiUnit(opq *big.Float) *big.Int {
	f := new(big.Float).Mul(opq, big.NewFloat(float64(TokenInWeiUnit)))
	wei, _ := f.Int(new(big.Int)) // ignore the accuracy
	return wei
}

/*ConvertFromWeiUnit converts wei unit to token unit */
func ConvertFromWeiUnit(wei *big.Int) *big.Float {
	weiInFloat := new(big.Float).SetInt(wei)
	return new(big.Float).Quo(weiInFloat, big.NewFloat(float64(TokenInWeiUnit)))
}

/*ConvertWeiToGwei converts from wei to gwei */
func ConvertWeiToGwei(wei *big.Int) *big.Int {
	return new(big.Int).Quo(wei, big.NewInt(params.GWei))
}

/*ConvertGweiToWei converts from gwei to wei */
func ConvertGweiToWei(gwei *big.Int) *big.Int {
	return new(big.Int).Mul(gwei, big.NewInt(params.GWei))
}
