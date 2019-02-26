package utils

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// a very big number that will lead to ErrTxnTooBig for write.
const guessedMaxBatchSize = 200000

var testDBID = []string{"prefix", "genHash", "data"}

func Test_KVStore_Init(t *testing.T) {
	err := InitKvStore()

	assert.Nil(t, err)
	defer CloseKvStore()
}

func Test_KVStore_MassBatchSet(t *testing.T) {
	InitKvStore()
	defer CloseKvStore()

	err := BatchSet(getKvPairs(guessedMaxBatchSize), TestValueTimeToLive)
	assert.Nil(t, err)

	kvs, _ := BatchGet(&KVKeys{strconv.Itoa(guessedMaxBatchSize - 1)})
	AssertTrue(len(*kvs) == 1, t, "Expect only an item")
}

func Test_KVStoreBatchGet(t *testing.T) {
	InitKvStore()
	defer CloseKvStore()

	BatchSet(&KVPairs{"key": "opacity"}, TestValueTimeToLive)

	kvs, err := BatchGet(&KVKeys{"key"})
	assert.Nil(t, err)

	AssertTrue(len(*kvs) == 1, t, "")
	assert.Nil(t, err)
}

func Test_KVStoreBatchGet_WithMissingKey(t *testing.T) {
	InitKvStore()
	defer CloseKvStore()

	BatchSet(&KVPairs{"key": "opacity"}, TestValueTimeToLive)

	kvs, err := BatchGet(&KVKeys{"key", "unknownKey"})
	assert.Nil(t, err)

	AssertTrue(len(*kvs) == 1, t, "")
	assert.Equal(t, "opacity", (*kvs)["key"])
}

func Test_KVStore_MassBatchGet(t *testing.T) {
	InitKvStore()
	defer CloseKvStore()

	err := BatchSet(getKvPairs(guessedMaxBatchSize), TestValueTimeToLive)
	assert.Nil(t, err)

	kvs, _ := BatchGet(getKeys(guessedMaxBatchSize))
	AssertTrue(len(*kvs) == guessedMaxBatchSize, t, "")
}

func Test_KVStoreBatchDelete(t *testing.T) {
	InitKvStore()
	defer CloseKvStore()

	BatchSet(&KVPairs{"key1": "opacity1", "key2": "opacity2"}, TestValueTimeToLive)

	err := BatchDelete(&KVKeys{"key1"})
	assert.Nil(t, err)

	kvs, err := BatchGet(&KVKeys{"key1"})
	assert.Nil(t, err)
	AssertTrue(len(*kvs) == 0, t, "")
}

func Test_KVStore_MassBatchDelete(t *testing.T) {
	InitKvStore()
	defer CloseKvStore()

	err := BatchSet(getKvPairs(guessedMaxBatchSize), TestValueTimeToLive)
	assert.Nil(t, err)

	err = BatchDelete(getKeys(guessedMaxBatchSize))
	assert.Nil(t, err)
}

func Test_KVStore_RemoveAllKvStoreData(t *testing.T) {
	InitKvStore()
	defer CloseKvStore()

	BatchSet(getKvPairs(2), TestValueTimeToLive)
	err := RemoveAllKvStoreData()
	assert.Nil(t, err)

	InitKvStore()
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
	for i := 0; i < guessedMaxBatchSize; i++ {
		keys = append(keys, strconv.Itoa(i))
	}
	return &keys
}
