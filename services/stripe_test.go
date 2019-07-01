package services

import (
	"testing"

	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go"
)

func Test_Stripe_Init(t *testing.T) {
	utils.SetTesting("../.env")
	err := InitStripe()
	assert.Nil(t, err)
}

func Test_CreateCharge(t *testing.T) {
	if utils.Env.StripeKey != utils.Env.StripeKeyTest {
		t.Fatalf("wrong stripe key")
		return
	}

	costInDollars := float64(15)
	stripeToken := RandTestStripeToken()

	_, err := CreateCharge(costInDollars, stripeToken)
	assert.Nil(t, err)
}

func Test_CheckChargeStatus(t *testing.T) {
	if utils.Env.StripeKey != utils.Env.StripeKeyTest {
		t.Fatalf("wrong stripe key")
		return
	}

	costInDollars := float64(15)
	stripeToken := RandTestStripeToken()

	c, _ := CreateCharge(costInDollars, stripeToken)

	status, err := CheckChargeStatus(c.ID)
	assert.Nil(t, err)
	assert.Equal(t, string(stripe.PaymentIntentStatusSucceeded), status)
}

func Test_CheckChargePaid(t *testing.T) {
	if utils.Env.StripeKey != utils.Env.StripeKeyTest {
		t.Fatalf("wrong stripe key")
		return
	}

	costInDollars := float64(15)
	stripeToken := RandTestStripeToken()

	c, _ := CreateCharge(costInDollars, stripeToken)

	paid, err := CheckChargePaid(c.ID)
	assert.Nil(t, err)
	assert.True(t, paid)
}
