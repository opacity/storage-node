package services

import (
	"testing"

	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go"
)

var testTokens = []string{
	"tok_visa",
	"tok_visa_debit",
	"tok_mastercard",
	"tok_mastercard_debit",
	"tok_mastercard_prepaid	",
	"tok_amex",
	"tok_discover",
	"tok_diners",
	"tok_jcb",
	"tok_unionpay",
}

func Test_Stripe_Init(t *testing.T) {
	utils.SetTesting("../.env")
	err := InitStripe()
	assert.Nil(t, err)
}

func randToken() string {
	return testTokens[utils.RandIndex(len(testTokens))]
}

func Test_CreateCharge(t *testing.T) {
	if utils.Env.StripeKey != utils.Env.StripeKeyTest {
		t.Fatalf("wrong stripe key")
		return
	}

	costInDollars := int64(15)
	stripeToken := randToken()

	_, err := CreateCharge(costInDollars, stripeToken)
	assert.Nil(t, err)
}

func Test_CheckChargeStatus(t *testing.T) {
	if utils.Env.StripeKey != utils.Env.StripeKeyTest {
		t.Fatalf("wrong stripe key")
		return
	}

	costInDollars := int64(15)
	stripeToken := randToken()

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

	costInDollars := int64(15)
	stripeToken := randToken()

	c, _ := CreateCharge(costInDollars, stripeToken)

	paid, err := CheckChargePaid(c.ID)
	assert.Nil(t, err)
	assert.True(t, paid)
}
