package utils

import (
	"errors"
	"io/ioutil"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/dgraph-io/badger"
)

const badgerDirProd = "/var/lib/badger/prod"

/*TestValueTimeToLive is some default value we can use in unit
tests for K:V pairs in badger*/
const TestValueTimeToLive = 1 * time.Minute
const BatchWriteMaxItems = 25
const BatchReadMaxItems = 100

// Singleton DB
var badgerDB *badger.DB
var dbNoInitError error
var badgerDirTest string

/*KVPairs is a type.  Map key strings to value strings*/
type KVPairs map[string]string

/*KVKeys is a type.  An array of key strings*/
type KVKeys []string

type DynamoMetadata struct {
	MetadataKey string `json:"MetadataKey" binding:"omitempty"`
	Value       string `json:"Value" binding:"omitempty"`
	TTL         int64  `json:"TTL" binding:"omitempty"`
}

func init() {
	dbNoInitError = errors.New("badgerDB not initialized, Call InitKvStore() first")

	badgerDirTest, _ = ioutil.TempDir("", "badgerForUnitTest")
}

/*InitKvStore returns db so that caller can call CloseKvStore to close it when it is done.*/
func InitKvStore() (err error) {
	if badgerDB != nil {
		return nil
	}

	// Setup opts
	var opts badger.Options

	if IsTestEnv() {
		opts = badger.DefaultOptions(badgerDirTest).WithTruncate(true)
	} else {
		opts = badger.DefaultOptions(badgerDirProd).WithTruncate(true)
	}

	badgerDB, err = badger.Open(opts)
	LogIfError(err, nil)
	return err
}

/*CloseKvStore closes the db.*/
func CloseKvStore() error {
	if badgerDB == nil {
		return nil
	}

	err := badgerDB.Close()
	LogIfError(err, nil)
	badgerDB = nil
	return err
}

// RemoveKvStore removes all the data along with the table
func RemoveKvStore() error {
	return DynamodbSvc.DeleteTable()
}

// GetValueFromKV gets a single value from the provided key
func GetValueFromKV(key string) (value string, expirationTime time.Time, err error) {
	expirationTime = time.Now()

	if key == "" {
		return value, expirationTime, errors.New("no key specified")
	}

	result := DynamoMetadata{}
	output, err := DynamodbSvc.Get("MetadataKey", key)
	if err != nil {
		return
	}

	err = dynamodbattribute.UnmarshalMap(output.Item, &result)
	if err != nil {
		return
	}
	value = result.Value
	expirationTime = time.Unix(int64(result.TTL), 0)

	return
}

// BatchGet returns KVPairs for a set of keys. It won't treat Key missing as error.
func BatchGet(ks *KVKeys) (kvs *KVPairs, err error) {
	kvs = &KVPairs{}
	batchKeys := make([]string, 0, BatchReadMaxItems)

	process := func() error {
		keys := []map[string]*dynamodb.AttributeValue{}
		// for _, k := range *ks {
		for _, k := range batchKeys {
			if k == "" {
				continue
			}
			keys = append(keys, map[string]*dynamodb.AttributeValue{
				"MetadataKey": {
					S: aws.String(k),
				},
			})
		}

		input := &dynamodb.BatchGetItemInput{
			RequestItems: map[string]*dynamodb.KeysAndAttributes{
				DynamodbSvc.tableName: {
					Keys: keys,
				},
			},
		}

		batchResult, err := DynamodbSvc.dynamodb.BatchGetItem(input)
		results := batchResult.Responses[DynamodbSvc.tableName]
		for _, result := range results {
			item := DynamoMetadata{}
			err = dynamodbattribute.UnmarshalMap(result, &item)
			if err != nil {
				return err
			}
			(*kvs)[item.MetadataKey] = item.Value
		}

		return nil
	}

	for k := range *kvs {
		batchKeys = append(batchKeys, k)
		if len(batchKeys) == BatchReadMaxItems {
			process()
			batchKeys = make([]string, 0, BatchWriteMaxItems)
		}
	}
	if len(batchKeys) > 0 {
		process()
	}

	LogIfError(err, map[string]interface{}{"batchSize": len(*ks)})

	return
}

// BatchSet updates a set of KVPairs. Return error if any fails.
func BatchSet(kvs *KVPairs, ttl time.Duration) error {
	ttl = getTTL(ttl)

	batchKeys := make([]string, 0, BatchWriteMaxItems)
	process := func() error {
		requests := []*dynamodb.WriteRequest{}

		for _, k := range batchKeys {
			if k == "" {
				return errors.New("object key empty")
			}
			dynamoItem := DynamoMetadata{
				MetadataKey: k,
				Value:       (*kvs)[k],
				TTL:         time.Now().Add(ttl).Unix(),
			}
			item, err := dynamodbattribute.MarshalMap(dynamoItem)
			if err != nil {
				return errors.New("object could not be created")
			}
			wr := dynamodb.WriteRequest{
				PutRequest: &dynamodb.PutRequest{
					Item: item,
				},
			}
			requests = append(requests, &wr)
		}

		err := DynamodbSvc.SetBatch(requests)
		if err != nil {
			return err
		}

		return nil
	}

	for k := range *kvs {
		batchKeys = append(batchKeys, k)
		if len(batchKeys) == BatchWriteMaxItems {
			process()
			batchKeys = make([]string, 0, BatchWriteMaxItems)
		}
	}
	if len(batchKeys) > 0 {
		process()
	}

	return nil
}

// BatchDelete deletes a set of KVKeys, Return error if any fails.
func BatchDelete(ks *KVKeys) error {
	var err error
	txn := badgerDB.NewTransaction(true)
	for _, key := range *ks {
		e := txn.Delete([]byte(key))
		if e == nil {
			continue
		}

		if e == badger.ErrTxnTooBig {
			e = nil
			if commitErr := txn.Commit(); commitErr != nil {
				e = commitErr
			} else {
				txn = badgerDB.NewTransaction(true)
				e = txn.Delete([]byte(key))
			}
		}

		if e != nil {
			err = e
			break
		}
	}

	defer txn.Discard()
	if err == nil {
		err = txn.Commit()
	}

	LogIfError(err, map[string]interface{}{"batchSize": len(*ks)})
	return err
}

func getTTL(ttl time.Duration) time.Duration {
	if !IsTestEnv() {
		return ttl
	}
	return TestValueTimeToLive
}
