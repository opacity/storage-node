package utils

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/dgraph-io/badger/pb"
)

// TestValueTimeToLive is some default value we can use
const TestValueTimeToLive = 1 * time.Minute
const BatchWriteMaxItems = 25
const BatchReadMaxItems = 100

type KVPairs map[string]string
type KVKeys []string
type DynamoMetadata struct {
	MetadataKey string `json:"MetadataKey" binding:"omitempty"`
	Value       string `json:"Value" binding:"omitempty"`
	TTL         int64  `json:"TTL" binding:"omitempty"`
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
		if err != nil {
			return err
		}
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

	for _, k := range *ks {
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

// BatchSetKV updates a set of KVPairs. Return error if any fails.
func BatchSetKV(list *pb.KVList) error {
	kvs := list.GetKv()
	batchKeys := make([]*pb.KV, 0, BatchWriteMaxItems)
	process := func() error {
		requests := []*dynamodb.WriteRequest{}

		for _, kv := range batchKeys {
			dynamoItem := DynamoMetadata{
				MetadataKey: string(kv.Key),
				Value:       string(kv.Value),
				TTL:         int64(kv.ExpiresAt),
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

	for _, k := range kvs {
		batchKeys = append(batchKeys, k)
		if len(batchKeys) == BatchWriteMaxItems {
			process()
			batchKeys = make([]*pb.KV, 0, BatchWriteMaxItems)
		}
	}
	if len(batchKeys) > 0 {
		process()
	}

	return nil
}

// BatchDelete deletes a set of KVKeys, Return error if any fails.
func BatchDelete(ks *KVKeys) error {
	batchKeys := make([]string, 0, BatchWriteMaxItems)
	process := func() error {
		requests := []*dynamodb.WriteRequest{}

		for _, k := range batchKeys {
			if k == "" {
				return errors.New("object key empty")
			}

			dr := dynamodb.WriteRequest{
				DeleteRequest: &dynamodb.DeleteRequest{
					Key: map[string]*dynamodb.AttributeValue{
						"MetadataKey": {
							S: aws.String(k),
						},
					},
				},
			}
			requests = append(requests, &dr)
		}

		err := DynamodbSvc.SetBatch(requests)
		if err != nil {
			return err
		}

		return nil
	}

	for _, k := range *ks {
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

func getTTL(ttl time.Duration) time.Duration {
	if !IsTestEnv() {
		return ttl
	}
	return TestValueTimeToLive
}
