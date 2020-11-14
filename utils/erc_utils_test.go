package utils

import (
	"math/big"
	"testing"
)

func Test_ConvertToWeiUnit(t *testing.T) {
	v := ConvertToWeiUnit(big.NewFloat(0.2))

	AssertTrue(v.Cmp(big.NewInt(200000000000000000)) == 0, t, "")
}

func Test_ConvertToWeiUnit_SmallValue(t *testing.T) {
	v := ConvertToWeiUnit(big.NewFloat(0.000000000000000002))

	AssertTrue(v.Cmp(big.NewInt(2)) == 0, t, "")
}

func Test_ConvertToWeiUnit_ConsiderAsZero(t *testing.T) {
	v := ConvertToWeiUnit(big.NewFloat(0.0000000000000000002))

	AssertTrue(v.Cmp(big.NewInt(0)) == 0, t, "")
}

func Test_ConvertToOpctUnit(t *testing.T) {
	v := ConvertFromWeiUnit(big.NewInt(200000000000000000))

	AssertTrue(v.String() == big.NewFloat(.2).String(), t, "")
}

func Test_ConvertToOpctUnit_SmallValue(t *testing.T) {
	v := ConvertFromWeiUnit(big.NewInt(2))

	AssertTrue(v.String() == big.NewFloat(.000000000000000002).String(), t, "")
}

func Test_ConvertGweiToWei(t *testing.T) {
	gwei := big.NewInt(1)
	expectedWei := big.NewInt(1000000000)

	weiResult := ConvertGweiToWei(gwei)
	AssertTrue(expectedWei.String() == weiResult.String(), t, "")
}

func Test_ConvertWeiToGwei(t *testing.T) {
	wei := big.NewInt(1000000000)
	expectedGwei := big.NewInt(1)

	gweiResult := ConvertWeiToGwei(wei)
	AssertTrue(expectedGwei.String() == gweiResult.String(), t, "")
}
