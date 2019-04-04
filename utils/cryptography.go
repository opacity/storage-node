package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
)


/*Encrypt encrypts a secret using a key and a nonce*/
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

/*EncryptWithErrorReturn encrypts a secret using a key and a nonce and returns an error if it fails*/
func EncryptWithErrorReturn(key string, secret string, nonce string) ([]byte, error) {
	keyInBytes, errDecodeKey := hex.DecodeString(key)
	secretInBytes, errDecodeSecret := hex.DecodeString(secret)
	block, errNewCipher := aes.NewCipher(keyInBytes)
	gcm, errNewGCM := cipher.NewGCM(block)
	nonceInBytes, errDecodeNonce := hex.DecodeString(nonce[0 : 2*gcm.NonceSize()])
	err := CollectErrors([]error{errDecodeKey, errDecodeSecret, errNewCipher, errNewGCM, errDecodeNonce})
	if err != nil {
		return []byte{}, err
	}
	data := gcm.Seal(nil, nonceInBytes, secretInBytes, nil)
	return data, nil
}

/*Decrypt decrypts a secret using a key and a nonce*/
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

/*DecryptWithErrorReturn decrypts a secret using a key and a nonce and returns an error if it fails*/
func DecryptWithErrorReturn(key string, cipherText string, nonce string) ([]byte, error) {
	keyInBytes, errDecodeKey := hex.DecodeString(key)
	cipherTextInBytes, cipherTextDecodeSecret := hex.DecodeString(cipherText)
	block, errNewCipher := aes.NewCipher(keyInBytes)
	gcm, errNewGCM := cipher.NewGCM(block)
	nonceInBytes, errDecodeNonce := hex.DecodeString(nonce[0 : 2*gcm.NonceSize()])
	data, decryptErr := gcm.Open(nil, nonceInBytes, cipherTextInBytes, nil)
	return data, CollectErrors([]error{
		errDecodeKey,
		cipherTextDecodeSecret,
		errNewCipher,
		errNewGCM,
		errDecodeNonce,
		decryptErr,
	})
}
