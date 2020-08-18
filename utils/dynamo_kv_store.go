package utils

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"strconv"
	"time"
)

type Metadata struct {
	Key            string `json:"key" dynamodbav:"key"`
	Value          string `json:"key" dynamodbav:"value"`
	ExpirationTime int64  `json:"expiration_time" dynamodbav:"expiration_time"`
}

var dynamoClient dynamodbiface.DynamoDBAPI

func InitDynamoKvStore() (err error) {
	dynamoClient = dynamodbiface.DynamoDBAPI(dynamodb.New(session.Must(session.NewSession()), aws.NewConfig().WithEndpoint(Env.DynamoEndpoint).WithRegion(Env.DynamoRegion)))
	return nil
}

func GetValueFromDynamoKv(key string) (value string, expirationTime time.Time, err error) {
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"key": {
				S: aws.String(key),
			},
		},
		ConsistentRead: aws.Bool(true),
		TableName: aws.String(Env.DynamoTable),
	}

	result, err := dynamoClient.GetItem(input)
	if newErr := dynamoError(err); newErr != nil {
		return "", time.Now(), newErr
	}

	metadata := Metadata{}

	err = dynamodbattribute.UnmarshalMap(result.Item, &metadata)
	if newErr := dynamoError(err); newErr != nil {
		return "", time.Now(), newErr
	}

	if metadata.Key == "" {
		return "", time.Now(), errors.New("key not found in dynamo")
	}

	i, err := strconv.ParseInt(strconv.FormatInt(metadata.ExpirationTime, 10), 10, 64)

	if newErr := dynamoError(err); newErr != nil {
		return "", time.Now(), newErr
	}

	return metadata.Value, time.Unix(i, 0), nil
}

func BatchGetFromDynamoKv(ks *KVKeys) (*KVPairs, error) {
	kvs := KVPairs{}
	var keys []map[string]*dynamodb.AttributeValue

	for _, key := range *ks {
		entry := make(map[string]*dynamodb.AttributeValue)
		entry["key"] = &dynamodb.AttributeValue{
			S: aws.String(key),
		}
		keys = append(keys, entry)
	}

	input := &dynamodb.BatchGetItemInput{
		RequestItems: map[string]*dynamodb.KeysAndAttributes{
			Env.DynamoTable: {
				Keys: keys,
			},
		},
	}

	result, err := dynamoClient.BatchGetItem(input)
	if newErr := dynamoError(err); newErr != nil {
		return &KVPairs{}, newErr
	}

	items, ok := result.Responses[Env.DynamoTable]

	if !ok {
		return &KVPairs{}, errors.New("not found")
	}

	for _, itemValue := range items {
		metadata := Metadata{}
		err = dynamodbattribute.UnmarshalMap(itemValue, &metadata)
		if newErr := dynamoError(err); newErr != nil {
			return &KVPairs{}, newErr
		}
		kvs[metadata.Key] = metadata.Value
	}

	//TODO: verify we did this correctly
	// batchGetItemOutput has this in it
	//	// A map of table name to a list of items. Each object in Responses consists
	//	// of a table name, along with a map of attribute data consisting of the data
	//	// type and attribute value.
	//	Responses map[string][]map[string]*AttributeValue

	// If there are a bunch of problems with this method just use GetValueFromDynamoKv in succession

	return &kvs, nil
}

func BatchSetToDynamoKv(kvs *KVPairs, ttl time.Duration) error {
	var errArray []error
	for key, value := range *kvs {
		expirationTime := time.Now().Add(ttl)

		metadata := Metadata{
			Key:            key,
			Value:          value,
			ExpirationTime: expirationTime.Unix(),
		}
		item, err := dynamodbattribute.MarshalMap(metadata)
		if newErr := dynamoError(err); newErr != nil {
			AppendIfError(newErr, &errArray)
		}

		input := &dynamodb.PutItemInput{
			Item:                   item,
			ReturnConsumedCapacity: aws.String("TOTAL"),
			TableName:              aws.String(Env.DynamoTable),
		}

		_, err = dynamoClient.PutItem(input)
		if newErr := dynamoError(err); newErr != nil {
			AppendIfError(newErr, &errArray)
		}
	}

	return CollectErrors(errArray)
}

func BatchDeleteFromDynamoKv(ks *KVKeys) error {
	var errArray []error
	for _, key := range *ks {
		input := &dynamodb.DeleteItemInput{
			Key: map[string]*dynamodb.AttributeValue{
				"key": {
					S: aws.String(key),
				},
			},
			TableName: aws.String(Env.DynamoTable),
		}
		_, err := dynamoClient.DeleteItem(input)
		AppendIfError(err, &errArray)
	}

	return CollectErrors(errArray)
}

func MigrateToDynamo(ttl time.Duration) error {
	expiration := time.Now().Add(ttl)
	expirationTime := expiration.Unix()

	err := badgerDB.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()

			var valBytes []byte
			err := item.Value(func(val []byte) error {
				if val == nil {
					valBytes = nil
				} else {
					valBytes = append([]byte{}, val...)
				}
				return nil
			})
			if err != nil {
				return err
			}

			metadata := Metadata{
				Key:            string(key),
				Value:          string(valBytes),
				ExpirationTime: expirationTime,
			}

			dynamoItem, err := dynamodbattribute.MarshalMap(metadata)
			if newErr := dynamoError(err); newErr != nil {
				return newErr
			}

			input := &dynamodb.PutItemInput{
				Item:                   dynamoItem,
				ReturnConsumedCapacity: aws.String("TOTAL"),
				TableName:              aws.String(Env.DynamoTable),
			}

			_, err = dynamoClient.PutItem(input)
			if newErr := dynamoError(err); newErr != nil {
				return newErr
			}
		}
		return nil
	})
	return err
}

func dynamoError(err error) error {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeTableNotFoundException, dynamodb.ErrCodeResourceNotFoundException, dynamodb.ErrCodeIndexNotFoundException:
				return errors.Errorf("%s - %s", "Not found", aerr.Error())
			case dynamodb.ErrCodeItemCollectionSizeLimitExceededException, dynamodb.ErrCodeConditionalCheckFailedException:
				return errors.Errorf("%s - %s", "Bad request", aerr.Error())
			case dynamodb.ErrCodeTransactionCanceledException, "ValidationException":
				return errors.Errorf("%s - %s", "Bad request", aerr.Error())
			default:
				return errors.Errorf("%s - %s", "Internal error", aerr.Error())
			}
			return err
		} else {
			return err
		}
	}
	return nil
}
