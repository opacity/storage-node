package routes

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"net/http"
	"net/http/httptest"

	"crypto/ecdsa"

	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func returnVerificationThatWillSucceed(t *testing.T, reqBody interface{}) verification {
	privateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	return setupVerificationWithPrivateKey(t, reqBody, privateKey)
}

func setupVerificationWithPrivateKey(t *testing.T, reqBody interface{}, privateKey *ecdsa.PrivateKey) verification {
	reqJSON, err := json.Marshal(reqBody)
	assert.Nil(t, err)
	hash := utils.Hash(reqJSON)

	assert.Nil(t, err)
	signature, err := utils.Sign(hash, privateKey)
	assert.Nil(t, err)

	verification := verification{
		Signature: hex.EncodeToString(signature),
		Address:   utils.PubkeyToAddress(privateKey.PublicKey).Hex(),
	}

	return verification
}

func returnVerificationThatWillFail(t *testing.T, reqBody interface{}) verification {
	reqJSON, err := json.Marshal(reqBody)
	assert.Nil(t, err)
	hash := utils.Hash(reqJSON)

	privateKeyToSignWith, err := utils.GenerateKey()
	assert.Nil(t, err)
	wrongPrivateKey, err := utils.GenerateKey()
	assert.Nil(t, err)
	signature, err := utils.Sign(hash, privateKeyToSignWith)
	assert.Nil(t, err)

	verification := verification{
		Signature: hex.EncodeToString(signature),
		Address:   utils.PubkeyToAddress(wrongPrivateKey.PublicKey).Hex(),
	}

	return verification
}

func testVerificationFailed(t *testing.T, w *httptest.ResponseRecorder) {
	// Check to see if the response was what you expected
	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusForbidden, w.Code)
	}

	assert.Contains(t, w.Body.String(), signatureDidNotMatchResponse)
}
