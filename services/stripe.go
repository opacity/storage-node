package services

import (
	"errors"

	"github.com/opacity/storage-node/utils"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/charge"
)

func InitStripe() error {
	if utils.Env.StripeKey == "" {
		return errors.New("must specify stripe keys in .env file")
	}
	stripe.Key = utils.Env.StripeKey
	return nil
}

func CreateCharge(costInDollars float64, stripeToken string) (*stripe.Charge, error) {
	cost := int64(costInDollars * 100)

	params := &stripe.ChargeParams{
		Amount:              stripe.Int64(cost),
		Currency:            stripe.String(string(stripe.CurrencyUSD)),
		Description:         stripe.String("File Storage"),
		StatementDescriptor: stripe.String("File Storage"),
	}
	params.SetSource(stripeToken)
	return charge.New(params)
}

func CheckChargeStatus(chargeID string) (string, error) {
	c, err := charge.Get(chargeID, nil)
	if err != nil {
		return "", err
	}
	return c.Status, nil
}

func CheckChargePaid(chargeID string) (bool, error) {
	c, err := charge.Get(chargeID, nil)
	if err != nil {
		return false, err
	}
	return c.Paid, nil
}
