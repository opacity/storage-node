package services

import (
	"os"
	"testing"

	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go"
)

func TestMain(m *testing.M) {
	utils.SetTesting("../.env")
	InitStripe()
	os.Exit(m.Run())
}

func Test_CreateCharge(t *testing.T) {
	t.Skip()
	if utils.Env.StripeKey != utils.Env.StripeKeyTest {
		t.Fatalf("wrong stripe key")
		return
	}

	costInDollars := float64(15)
	stripeToken := RandTestStripeToken()

	_, err := CreateCharge(costInDollars, stripeToken, utils.RandHexString(64))
	assert.Nil(t, err)
}

func Test_CheckChargeStatus(t *testing.T) {
	t.Skip()
	if utils.Env.StripeKey != utils.Env.StripeKeyTest {
		t.Fatalf("wrong stripe key")
		return
	}

	costInDollars := float64(15)
	stripeToken := RandTestStripeToken()

	c, _ := CreateCharge(costInDollars, stripeToken, utils.RandHexString(64))

	status, err := CheckChargeStatus(c.ID)
	assert.Nil(t, err)
	assert.Equal(t, string(stripe.PaymentIntentStatusSucceeded), status)
}

func Test_CheckChargePaid(t *testing.T) {
	t.Skip()
	if utils.Env.StripeKey != utils.Env.StripeKeyTest {
		t.Fatalf("wrong stripe key")
		return
	}

	costInDollars := float64(15)
	stripeToken := RandTestStripeToken()

	c, _ := CreateCharge(costInDollars, stripeToken, utils.RandHexString(64))

	paid, err := CheckChargePaid(c.ID)
	assert.Nil(t, err)
	assert.True(t, paid)
}
