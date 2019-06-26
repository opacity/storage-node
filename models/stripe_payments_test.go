package models

import (
	"testing"

	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

func returnValidStripePaymentForTest() StripePayment {
	account := returnValidAccount()

	// Add account to DB
	DB.Create(&account)

	return StripePayment{
		StripeToken: services.RandTestStripeToken(),
		AccountID:   account.AccountID,
	}
}

func Test_Init_Stripe_Payments(t *testing.T) {
	utils.SetTesting("../.env")
	Connect(utils.Env.TestDatabaseURL)
}

func Test_Valid_Stripe_Payment_Passes(t *testing.T) {
	stripePayment := returnValidStripePaymentForTest()

	if err := DB.Create(&stripePayment).Error; err != nil {
		t.Fatalf("should have created row but didn't: " + err.Error())
	}
}

func Test_Valid_Stripe_Fails_If_No_Account_Exists(t *testing.T) {
	stripePayment := returnValidStripePaymentForTest()
	account, _ := GetAccountById(stripePayment.AccountID)
	DB.Delete(&account)

	if err := DB.Create(&stripePayment).Error; err == nil {
		t.Fatalf("row creation should have failed")
	}
}
