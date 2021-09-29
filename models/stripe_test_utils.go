package models

import (
	"math/rand"
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

func RandTestStripeToken() string {
	return testTokens[rand.Intn(len(testTokens))]
}
