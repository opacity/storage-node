package models

import (
	"math/rand"
	"testing"

	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go"
)

var testTokens = []string{
	`tok_visa`,
	`tok_visa_debit`,
	`tok_mastercard`,
	`tok_mastercard_debit`,
	`tok_mastercard_prepaid`,
	`tok_amex`,
	`tok_discover`,
	`tok_diners`,
	`tok_jcb`,
	`tok_unionpay`,
}

func Test_Stripe_Init(t *testing.T) {
	utils.SetTesting("../.env")
}

func Test_CreateCharge(t *testing.T) {
	if utils.Env.StripeKey != utils.Env.StripeKeyTest {
		t.Fatalf("wrong stripe key")
		return
	}

	costInDollars := float64(15)
	stripeToken := RandTestStripeToken()

	_, err := services.CreateCharge(costInDollars, stripeToken, utils.RandHexString(64))
	assert.Nil(t, err)
}

func Test_CheckChargeStatus(t *testing.T) {
	if utils.Env.StripeKey != utils.Env.StripeKeyTest {
		t.Fatalf("wrong stripe key")
		return
	}

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

func RandTestStripeToken() string {
	return testTokens[rand.Intn(len(testTokens))]
}
