package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
)

func Encrypt(key string, secret string, nonce string) []byte {
	keyInBytes, err := hex.DecodeString(key)
	PanicOnError(err)
	secretInBytes, err := hex.DecodeString(secret)
	PanicOnError(err)
	block, err := aes.NewCipher(keyInBytes)
	PanicOnError(err)
	gcm, err := cipher.NewGCM(block)
	PanicOnError(err)
	nonceInBytes, err := hex.DecodeString(nonce[0 : 2*gcm.NonceSize()])
	PanicOnError(err)
	data := gcm.Seal(nil, nonceInBytes, secretInBytes, nil)
	return data
}

func Decrypt(key string, cipherText string, nonce string) []byte {
	keyInBytes, err := hex.DecodeString(key)
	PanicOnError(err)
	data, err := hex.DecodeString(cipherText)
	PanicOnError(err)
	block, err := aes.NewCipher(keyInBytes)
	PanicOnError(err)
	gcm, err := cipher.NewGCM(block)
	PanicOnError(err)
	nonceInBytes, err := hex.DecodeString(nonce[0 : 2*gcm.NonceSize()])
	PanicOnError(err)
	data, err = gcm.Open(nil, nonceInBytes, data, nil)
	if err != nil {
		return nil
	}
	return data
}
