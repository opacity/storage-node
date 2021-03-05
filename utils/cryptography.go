package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const SigLengthInBytes = 64

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

/*HashString hashes input string arguments and outputs a hash encoded as a hex string*/
func HashString(dataString string) (string, error) {
	dataBytes, err := hex.DecodeString(dataString)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(crypto.Keccak256(dataBytes)), nil
}

/*HashStringV2 hashes input string arguments and outputs a hash encoded as a base64 string*/
func HashStringV2(data []byte) string {
	return base64.StdEncoding.EncodeToString(crypto.Keccak256(data))
}

/*Hash hashes input byte arguments*/
func Hash(data ...[]byte) []byte {
	return crypto.Keccak256(data...)
}

/*Recover recovers a public key from an ECDSA sig*/
func Recover(hash []byte, sig []byte) (*ecdsa.PublicKey, error) {
	return crypto.SigToPub(hash, sig)
}

/*Verify verifies that a particular key was the signer of a message*/
func Verify(pubKey []byte, hash []byte, sig []byte) (bool, error) {
	if len(sig) > SigLengthInBytes {
		sig = sig[:SigLengthInBytes]
	}
	return crypto.VerifySignature(pubKey, hash, sig), nil
}

/*VerifyFromStrings verifies that a particular key was the signer of a message, with hex strings
as arguments*/
func VerifyFromStrings(publicKey string, hash string, sig string) (bool, error) {
	publicKeyBytes, err := hex.DecodeString(publicKey)
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

	return Verify(publicKeyBytes, hashBytes, sigBytes)
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

/*PubkeyToHex takes a public key and converts it to a hex string*/
func PubkeyToHex(p ecdsa.PublicKey) string {
	return hex.EncodeToString(crypto.FromECDSAPub(&p))
}

/*PubkeyCompressedToHex takes a public key, compresses it and converts it to a hex string*/
func PubkeyCompressedToHex(p ecdsa.PublicKey) string {
	compressed := crypto.CompressPubkey(&p)
	return hex.EncodeToString(compressed)
}
