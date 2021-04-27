package utils

import (
	"strconv"
	"testing"

	"time"

	"github.com/stretchr/testify/assert"
)

const MaxBatchSize = 10000

func Test_KVStore_Init(t *testing.T) {
	SetTesting("../.env")
}

func Test_KVStore_MassBatchSet(t *testing.T) {
	SetTesting("../.env")
	err := BatchSet(getKvPairs(MaxBatchSize), TestValueTimeToLive)
	assert.Nil(t, err)

	kvs, _ := BatchGet(&KVKeys{strconv.Itoa(MaxBatchSize - 1)})
	AssertTrue(len(*kvs) == 1, t, "Expect only an item")
}

func Test_KVStoreGetValueFromKV(t *testing.T) {
	SetTesting("../.env")
	key := "key"
	valueSet := "opacity"

	BatchSet(&KVPairs{key: valueSet}, TestValueTimeToLive)

	value, expirationTime, err := GetValueFromKV(key)
	assert.Nil(t, err)
	assert.Equal(t, valueSet, value)
	assert.Equal(t, time.Now().Add(TestValueTimeToLive).Second(), expirationTime.Second())
	assert.Equal(t, time.Now().Add(TestValueTimeToLive).Minute(), expirationTime.Minute())
	assert.Equal(t, time.Now().Add(TestValueTimeToLive).Hour(), expirationTime.Hour())
	assert.Equal(t, time.Now().Add(TestValueTimeToLive).Day(), expirationTime.Day())
	assert.Equal(t, time.Now().Add(TestValueTimeToLive).Month(), expirationTime.Month())
	assert.Equal(t, time.Now().Add(TestValueTimeToLive).Year(), expirationTime.Year())
}

func Test_KVStoreBatchGet(t *testing.T) {
	SetTesting("../.env")
	BatchSet(&KVPairs{"key": "opacity"}, TestValueTimeToLive)

	kvs, err := BatchGet(&KVKeys{"key"})
	assert.Nil(t, err)

	AssertTrue(len(*kvs) == 1, t, "")
	assert.Nil(t, err)
}

func Test_KVStoreBatchGet_WithMissingKey(t *testing.T) {
	SetTesting("../.env")
	BatchSet(&KVPairs{"key": "opacity"}, TestValueTimeToLive)

	kvs, err := BatchGet(&KVKeys{"key", "unknownKey"})
	assert.Nil(t, err)

	AssertTrue(len(*kvs) == 1, t, "")
	assert.Equal(t, "opacity", (*kvs)["key"])
}

func Test_KVStore_MassBatchGet(t *testing.T) {
	SetTesting("../.env")
	err := BatchSet(getKvPairs(MaxBatchSize), TestValueTimeToLive)
	assert.Nil(t, err)

	kvs, _ := BatchGet(getKeys(MaxBatchSize))
	AssertTrue(len(*kvs) == MaxBatchSize, t, "")
}

func Test_KVStoreBatchDelete(t *testing.T) {
	SetTesting("../.env")
	BatchSet(&KVPairs{"key1": "opacity1", "key2": "opacity2"}, TestValueTimeToLive)

	err := BatchDelete(&KVKeys{"key1"})
	assert.Nil(t, err)

	kvs, err := BatchGet(&KVKeys{"key1"})
	assert.Nil(t, err)
	AssertTrue(len(*kvs) == 0, t, "")
}

func Test_KVStore_MassBatchDelete(t *testing.T) {
	SetTesting("../.env")
	err := BatchSet(getKvPairs(MaxBatchSize), TestValueTimeToLive)
	assert.Nil(t, err)

	err = BatchDelete(getKeys(MaxBatchSize))
	assert.Nil(t, err)
}

func Test_KVStore_RemoveAllKvStoreData(t *testing.T) {
	SetTesting("../.env")
	BatchSet(getKvPairs(2), TestValueTimeToLive)
	err := RemoveKvStore()
	assert.Nil(t, err)

	SetTesting("../.env")
	kvs, _ := BatchGet(getKeys(2))

	AssertTrue(len(*kvs) == 0, t, "")
}

func getKvPairs(count int) *KVPairs {
	pairs := KVPairs{}
	for i := 0; i < count; i++ {
		pairs[strconv.Itoa(i)] = strconv.Itoa(i)
	}
	return &pairs
}

func getKeys(count int) *KVKeys {
	keys := KVKeys{}
	for i := 0; i < MaxBatchSize; i++ {
		keys = append(keys, strconv.Itoa(i))
	}
	return &keys
}
