package services

import (
	"os"
	"testing"

	"github.com/opacity/storage-node/utils"
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

func TestMain(m *testing.M) {
	utils.SetTesting("../.env")
	InitStripe()
	os.Exit(m.Run())
}

func RandTestStripeToken() string {
	return testTokens[utils.RandIndex(len(testTokens))]
}
