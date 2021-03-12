package routes

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/dag"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Init_MetadataV2(t *testing.T) {

	setupTests(t)
}

func Test_GetMetadataV2Handler_Returns_MetadataV2(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataV2Key := utils.GenerateFileHandle()
	testMetadataV2Value := utils.GenerateFileHandle()

	if err := utils.BatchSet(&utils.KVPairs{testMetadataV2Key: testMetadataV2Value}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	getMetadataV2 := metadataV2KeyObject{
		MetadataV2Key: testMetadataV2Key,
		Timestamp:     time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getMetadataV2)

	get := metadataV2KeyReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreatePaidAccountForTest(t, accountID)

	w := httpPostRequestHelperForTest(t, MetadataGetPath, "v2", get)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), testMetadataV2Value)
}

func Test_GetMetadataV2Handler_Error_If_Not_Paid(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	testMetadataV2Key := utils.GenerateFileHandle()
	testMetadataV2Value := utils.GenerateFileHandle()

	if err := utils.BatchSet(&utils.KVPairs{testMetadataV2Key: testMetadataV2Value}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	getMetadataV2 := metadataV2KeyObject{
		MetadataV2Key: testMetadataV2Key,
		Timestamp:     time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getMetadataV2)

	get := metadataV2KeyReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreateUnpaidAccountForTest(t, accountID)

	w := httpPostRequestHelperForTest(t, MetadataV2GetPath, "v2", get)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `"invoice"`)
}

func Test_GetMetadataV2Handler_Error_If_Not_In_KV_Store(t *testing.T) {
	testMetadataV2Key := utils.GenerateFileHandle()

	getMetadataV2 := metadataV2KeyObject{
		MetadataV2Key: testMetadataV2Key,
		Timestamp:     time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, getMetadataV2)

	get := metadataV2KeyReq{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreatePaidAccountForTest(t, accountID)

	w := httpPostRequestHelperForTest(t, MetadataV2GetPath, "v2", get)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func Test_UpdateMetadataV2Handler_Can_Update_MetadataV2(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	d := dag.NewDAG()
	vert := dag.NewDAGVertex(utils.RandByteSlice(32))

	d.AddReduced(*vert)

	testMetadataV2Key := utils.GenerateFileHandle()
	testMetadataV2Value := utils.GenerateFileHandle()
	newVertex := base64.StdEncoding.EncodeToString(vert.Binary())

	digest, _ := d.Digest(vert.ID, func(b []byte) []byte { s := sha256.Sum256(b); return s[:] })
	testKey, _ := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	testSig, _ := secp256k1.Sign(digest, testKey.X.Bytes())

	updateMetadataV2Obj := updateMetadataV2Object{
		MetadataV2Key:    testMetadataV2Key,
		MetadataV2Vertex: newVertex,
		MetadataV2Edges:  []string{},
		MetadataV2Sig:    base64.StdEncoding.EncodeToString(testSig),
		Timestamp:        time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, updateMetadataV2Obj)

	post := updateMetadataV2Req{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)
	err := account.IncrementMetadataCount()
	assert.Nil(t, err)
	err = account.UpdateMetadataSizeInBytes(0, int64(len(testMetadataV2Value)))
	assert.Nil(t, err)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	permissionHash, err := getPermissionHash(v.PublicKey, testMetadataV2Key, c)
	assert.Nil(t, err)

	permissionHashKey := getPermissionHashKeyForBadger(testMetadataV2Key)

	if err := utils.BatchSet(&utils.KVPairs{
		testMetadataV2Key: testMetadataV2Value,
		permissionHashKey: permissionHash,
	}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	w := httpPostRequestHelperForTest(t, MetadataV2AddPath, "v2", post)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), newVertex)

	metadataV2, _, _ := utils.GetValueFromKV(testMetadataV2Key)
	assert.Equal(t, base64.RawStdEncoding.EncodeToString(d.Binary()), metadataV2)

	accountFromDB, _ := models.GetAccountById(account.AccountID)
	assert.Equal(t, int64(len(d.Binary())), accountFromDB.TotalMetadataSizeInBytes)
}

func Test_UpdateMetadataV2Handler_Error_If_Not_Paid(t *testing.T) {
	ttl := utils.TestValueTimeToLive

	d := dag.NewDAG()
	vert := dag.NewDAGVertex(utils.RandByteSlice(32))

	d.AddReduced(*vert)

	testMetadataV2Key := utils.GenerateFileHandle()
	testMetadataV2Value := utils.GenerateFileHandle()
	newVertex := base64.StdEncoding.EncodeToString(vert.Binary())

	digest, _ := d.Digest(vert.ID, func(b []byte) []byte { s := sha256.Sum256(b); return s[:] })
	testKey, _ := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	testSig, _ := secp256k1.Sign(digest, testKey.X.Bytes())

	if err := utils.BatchSet(&utils.KVPairs{testMetadataV2Key: testMetadataV2Value}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	updateMetadataV2Obj := updateMetadataV2Object{
		MetadataV2Key:    testMetadataV2Key,
		MetadataV2Vertex: newVertex,
		MetadataV2Edges:  []string{},
		MetadataV2Sig:    base64.StdEncoding.EncodeToString(testSig),
		Timestamp:        time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, updateMetadataV2Obj)

	post := updateMetadataV2Req{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreateUnpaidAccountForTest(t, accountID)

	w := httpPostRequestHelperForTest(t, MetadataV2AddPath, "v2", post)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `"invoice"`)
}

func Test_UpdateMetadataV2Handler_Error_If_Key_Does_Not_Exist(t *testing.T) {
	d := dag.NewDAG()
	vert := dag.NewDAGVertex(utils.RandByteSlice(32))

	d.AddReduced(*vert)

	testMetadataV2Key := utils.GenerateFileHandle()
	newVertex := base64.StdEncoding.EncodeToString(vert.Binary())

	digest, _ := d.Digest(vert.ID, func(b []byte) []byte { s := sha256.Sum256(b); return s[:] })
	testKey, _ := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	testSig, _ := secp256k1.Sign(digest, testKey.X.Bytes())

	updateMetadataV2Obj := updateMetadataV2Object{
		MetadataV2Key:    testMetadataV2Key,
		MetadataV2Vertex: newVertex,
		MetadataV2Edges:  []string{},
		MetadataV2Sig:    base64.StdEncoding.EncodeToString(testSig),
		Timestamp:        time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, updateMetadataV2Obj)

	post := updateMetadataV2Req{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreatePaidAccountForTest(t, accountID)

	w := httpPostRequestHelperForTest(t, MetadataV2AddPath, "v2", post)
	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func Test_UpdateMetadataV2Handler_Error_If_Verification_Fails(t *testing.T) {
	d := dag.NewDAG()
	vert := dag.NewDAGVertex(utils.RandByteSlice(32))

	d.AddReduced(*vert)

	testMetadataV2Key := utils.GenerateFileHandle()
	newVertex := base64.StdEncoding.EncodeToString(vert.Binary())

	digest, _ := d.Digest(vert.ID, func(b []byte) []byte { s := sha256.Sum256(b); return s[:] })
	testKey, _ := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	testSig, _ := secp256k1.Sign(digest, testKey.X.Bytes())

	updateMetadataV2Obj := updateMetadataV2Object{
		MetadataV2Key:    testMetadataV2Key,
		MetadataV2Vertex: newVertex,
		MetadataV2Edges:  []string{},
		MetadataV2Sig:    base64.StdEncoding.EncodeToString(testSig),
		Timestamp:        time.Now().Unix(),
	}

	v, b, _, _ := returnInvalidVerificationAndRequestBody(t, updateMetadataV2Obj)

	post := updateMetadataV2Req{
		verification: v,
		requestBody:  b,
	}

	accountID, _ := utils.HashString(v.PublicKey)
	CreatePaidAccountForTest(t, accountID)

	w := httpPostRequestHelperForTest(t, MetadataV2AddPath, "v2", post)

	confirmVerifyFailedForTest(t, w)
}

func Test_Delete_MetadataV2_Fails_If_Unpaid(t *testing.T) {
	testMetadataV2Key := utils.RandSeqFromRunes(64, []rune("abcdef01234567890"))

	deleteMetadataV2Obj := metadataV2KeyObject{
		MetadataV2Key: testMetadataV2Key,
		Timestamp:     time.Now().Unix(),
	}

	models.BackendManager.CheckIfPaid = func(address common.Address, amount *big.Int) (bool, error) {
		return false, nil
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, deleteMetadataV2Obj)

	post := metadataV2KeyReq{
		verification: v,
		requestBody: requestBody{
			RequestBody: b.RequestBody,
		},
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreateUnpaidAccountForTest(t, accountID)
	account.TotalFolders = 1
	err := models.DB.Save(&account).Error
	assert.Nil(t, err)

	w := httpPostRequestHelperForTest(t, MetadataV2DeletePath, "v2", post)

	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "invoice")
}

func Test_Delete_MetadataV2_Fails_If_Permission_Hash_Does_Not_Match(t *testing.T) {
	testMetadataV2Key := utils.RandSeqFromRunes(64, []rune("abcdef01234567890"))
	testMetadataV2Value := "someValue"

	deleteMetadataV2Obj := metadataV2KeyObject{
		MetadataV2Key: testMetadataV2Key,
		Timestamp:     time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, deleteMetadataV2Obj)

	post := metadataV2KeyReq{
		verification: v,
		requestBody: requestBody{
			RequestBody: b.RequestBody,
		},
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)
	account.TotalFolders = 1
	account.TotalMetadataSizeInBytes = int64(len(testMetadataV2Value))
	err := models.DB.Save(&account).Error
	assert.Nil(t, err)

	permissionHashKey := getPermissionHashKeyForBadger(testMetadataV2Key)

	ttl := time.Until(account.ExpirationDate())

	if err := utils.BatchSet(&utils.KVPairs{
		testMetadataV2Key: testMetadataV2Value,
		permissionHashKey: "someIncorrectPermissionHash",
	}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	w := httpPostRequestHelperForTest(t, MetadataV2DeletePath, "v2", post)

	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), notAuthorizedResponse)
}

func Test_Delete_MetadataV2_Success(t *testing.T) {
	testMetadataV2Key := utils.RandSeqFromRunes(64, []rune("abcdef01234567890"))
	testMetadataV2Value := "someValue"

	deleteMetadataV2Obj := metadataV2KeyObject{
		MetadataV2Key: testMetadataV2Key,
		Timestamp:     time.Now().Unix(),
	}

	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, deleteMetadataV2Obj)

	post := metadataV2KeyReq{
		verification: v,
		requestBody: requestBody{
			RequestBody: b.RequestBody,
		},
	}

	accountID, _ := utils.HashString(v.PublicKey)
	account := CreatePaidAccountForTest(t, accountID)
	account.TotalFolders = 1
	account.TotalMetadataSizeInBytes = int64(len(testMetadataV2Value))
	err := models.DB.Save(&account).Error
	assert.Nil(t, err)

	accountFromDB, _ := models.GetAccountById(account.AccountID)
	assert.Equal(t, int64(len(testMetadataV2Value)), accountFromDB.TotalMetadataSizeInBytes)
	assert.Equal(t, 1, accountFromDB.TotalFolders)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	permissionHash, err := getPermissionHash(v.PublicKey, testMetadataV2Key, c)
	permissionHashKey := getPermissionHashKeyForBadger(testMetadataV2Key)

	ttl := time.Until(account.ExpirationDate())

	if err := utils.BatchSet(&utils.KVPairs{
		testMetadataV2Key: testMetadataV2Value,
		permissionHashKey: permissionHash,
	}, ttl); err != nil {
		t.Fatalf("there should not have been an error")
	}

	w := httpPostRequestHelperForTest(t, MetadataV2DeletePath, "v2", post)

	// Check to see if the response was what you expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), metadataV2DeletedRes.Status)
	accountFromDB, _ = models.GetAccountById(account.AccountID)
	assert.Equal(t, int64(0), accountFromDB.TotalMetadataSizeInBytes)
	assert.Equal(t, 0, accountFromDB.TotalFolders)
}
