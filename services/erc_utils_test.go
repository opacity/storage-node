package services

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ConvertToWeiUnit(t *testing.T) {
	v := ConvertToWeiUnit(big.NewFloat(0.2))

	assert.True(t, v.Cmp(big.NewInt(200000000000000000)) == 0)
}

func Test_ConvertToWeiUnit_SmallValue(t *testing.T) {
	v := ConvertToWeiUnit(big.NewFloat(0.000000000000000002))

	assert.True(t, v.Cmp(big.NewInt(2)) == 0)
}

func Test_ConvertToWeiUnit_ConsiderAsZero(t *testing.T) {
	v := ConvertToWeiUnit(big.NewFloat(0.0000000000000000002))

	assert.True(t, v.Cmp(big.NewInt(0)) == 0)
}

func Test_ConvertToOpctUnit(t *testing.T) {
	v := ConvertFromWeiUnit(big.NewInt(200000000000000000))

	assert.True(t, v.String() == big.NewFloat(.2).String())
}

func Test_ConvertToOpctUnit_SmallValue(t *testing.T) {
	v := ConvertFromWeiUnit(big.NewInt(2))

	assert.True(t, v.String() == big.NewFloat(.000000000000000002).String())
}

func Test_ConvertGweiToWei(t *testing.T) {
	gwei := big.NewInt(1)
	expectedWei := big.NewInt(1000000000)

	weiResult := ConvertGweiToWei(gwei)
	assert.True(t, expectedWei.String() == weiResult.String())
}

func Test_ConvertWeiToGwei(t *testing.T) {
	wei := big.NewInt(1000000000)
	expectedGwei := big.NewInt(1)

	gweiResult := ConvertWeiToGwei(wei)
	assert.True(t, expectedGwei.String() == gweiResult.String())
}
