package models

import (
	"testing"

	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go"
)

func Test_Stripe_Init(t *testing.T) {
	utils.SetTesting("../.env")
	SetTestPlans()
}

func Test_CreateCharge(t *testing.T) {
	costInDollars := float64(15)
	stripeToken := RandTestStripeToken()

	_, err := services.CreateCharge(costInDollars, stripeToken, utils.RandHexString(64))
	assert.Nil(t, err)
}

func Test_CheckChargeStatus(t *testing.T) {
	costInDollars := float64(15)
	stripeToken := RandTestStripeToken()

	c, _ := services.CreateCharge(costInDollars, stripeToken, utils.RandHexString(64))

	status, err := services.CheckChargeStatus(c.ID)
	assert.Nil(t, err)
	assert.Equal(t, string(stripe.PaymentIntentStatusSucceeded), status)
}

func Test_CheckChargePaid(t *testing.T) {
	costInDollars := float64(15)
	stripeToken := RandTestStripeToken()

	c, _ := services.CreateCharge(costInDollars, stripeToken, utils.RandHexString(64))

	paid, err := services.CheckChargePaid(c.ID)
	assert.Nil(t, err)
	assert.True(t, paid)
}
