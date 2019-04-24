package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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

/*Hash hashes input byte arguments*/
func Hash(data ...[]byte) []byte {
	return crypto.Keccak256(data...)
}

/*Recover recovers a public key from an ECDSA sig*/
func Recover(hash []byte, sig []byte) (*ecdsa.PublicKey, error) {
	return crypto.SigToPub(hash, sig)
}

/*Verify recovers a public key and checks it against an existing, known public key, and returns true if they match*/
func Verify(address []byte, hash []byte, sig []byte) (bool, error) {
	pubkey, err := Recover(hash, sig)
	addr := PubkeyToAddress(*pubkey)

	return bytes.Equal(address, addr[:]), err
}

/*VerifyFromStrings recovers a public key and checks it against an existing, known public key, and returns true if they match*/
func VerifyFromStrings(address string, hash string, sig string) (bool, error) {
	addressBytes, err := hex.DecodeString(address)
	if err != nil {
		return false, err
	}

	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		return false, err
	}

	sigBytes, err := hex.DecodeString(sig)
	if err != nil {
		return false, err
	}

	return Verify(addressBytes, hashBytes, sigBytes)
}

/*Sign signs a message with a private key*/
func Sign(msg []byte, prv *ecdsa.PrivateKey) ([]byte, error) {
	return crypto.Sign(msg, prv)
}

/*GenerateKey generates a random ecdsa private key*/
func GenerateKey() (*ecdsa.PrivateKey, error) {
	return crypto.GenerateKey()
}

/*PubkeyToAddress takes a public key and converts it to an ethereum public address*/
func PubkeyToAddress(p ecdsa.PublicKey) common.Address {
	return crypto.PubkeyToAddress(p)
}
