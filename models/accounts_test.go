package models

import (
	"testing"

	"time"

	"encoding/hex"

	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

var testNonce = "23384a8eabc4a4ba091cfdbcb3dbacdc27000c03e318fd52accb8e2380f11320"

func returnValidAccount() Account {
	ethAddress, privateKey, _ := services.EthWrapper.GenerateWallet()

	return Account{
		UserID:          utils.RandSeqFromRunes(64, []rune("abcdefg01234567890")),
		ExpirationDate:  time.Now().Add(1 * time.Minute),
		StorageLocation: "https://someFileStoragePlace.com/12345",
		StorageLimit:    BasicStorageLimit,
		PIN:             1234,
		PaymentStatus:   InitialPaymentInProgress,
		EthAddress:      ethAddress.String(),
		EthPrivateKey:   hex.EncodeToString(utils.Encrypt(utils.Env.EncryptionKey, privateKey, testNonce)),
	}
}

func Test_Init_Accounts(t *testing.T) {
	utils.SetTesting("../.env")
}

func Test_Valid_Account_Passes(t *testing.T) {
	account := returnValidAccount()

	if err := Validator.Struct(account); err != nil {
		t.Fatalf("account should have passed validation but didn't: " + err.Error())
	}
}

func Test_Empty_UserID_Fails(t *testing.T) {
	account := returnValidAccount()
	account.UserID = ""

	if err := Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_Invalid_UserID_Length_Fails(t *testing.T) {
	account := returnValidAccount()
	account.UserID = utils.RandSeqFromRunes(63, []rune("abcdefg01234567890"))

	if err := Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}

	account.UserID = utils.RandSeqFromRunes(65, []rune("abcdefg01234567890"))

	if err := Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_ExpirationDate_In_The_Past_Fails(t *testing.T) {
	account := returnValidAccount()
	account.ExpirationDate = time.Now().Add(-1 * time.Minute)

	if err := Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_StorageLocation_Invalid_URL_Fails(t *testing.T) {
	account := returnValidAccount()
	account.StorageLocation = "wrong"

	if err := Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_StorageLimit_Less_Than_100_Fails(t *testing.T) {
	account := returnValidAccount()
	account.StorageLimit = 99

	if err := Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_No_PIN_Fails(t *testing.T) {
	account := returnValidAccount()
	account.PIN = 0

	if err := Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_No_Eth_Address_Fails(t *testing.T) {
	account := returnValidAccount()
	account.EthAddress = ""

	if err := Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_Eth_Address_Invalid_Length_Fails(t *testing.T) {
	account := returnValidAccount()
	account.EthAddress = utils.RandSeqFromRunes(6, []rune("abcdefg01234567890"))

	if err := Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_No_Eth_Private_Key_Fails(t *testing.T) {
	account := returnValidAccount()
	account.EthPrivateKey = ""

	if err := Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_Eth_Private_Key_Invalid_Length_Fails(t *testing.T) {
	account := returnValidAccount()
	account.EthPrivateKey = utils.RandSeqFromRunes(6, []rune("abcdefg01234567890"))

	if err := Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}

func Test_No_Payment_Status_Fails(t *testing.T) {
	account := returnValidAccount()
	account.PaymentStatus = 0

	if err := Validator.Struct(account); err == nil {
		t.Fatalf("account should have failed validation")
	}
}
