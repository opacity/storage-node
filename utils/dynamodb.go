package utils

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var ErrDynamoDBKeyNotFound = errors.New("key does not exist")

type DynamodbWrapper struct {
	dynamodb            *dynamodb.DynamoDB
	tableName           string
	badgerMigrationDone bool
}

func NewDynamoDBSession(tableName string, region string, endpoint string) (*DynamodbWrapper, error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	awsConfig := aws.NewConfig()
	tagValue := Env.GoEnv
	endpoint = strings.ReplaceAll(endpoint, "\"", "")
	if IsDebugEnv() {
		awsConfig = awsConfig.WithEndpoint(endpoint).WithLogLevel(aws.LogDebugWithHTTPBody)
	}
	if IsTestEnv() {
		awsConfig.WithEndpoint(endpoint).WithLogLevel(aws.LogOff)
	}

	dynamodbInstance := dynamodb.New(sess, awsConfig)

	if err := CreateTable(tagValue, tableName, dynamodbInstance); err != nil {
		return nil, err
	}

	return &DynamodbWrapper{
		dynamodb:            dynamodbInstance,
		tableName:           tableName,
		badgerMigrationDone: false,
	}, nil
}

func (dynamodbSvc *DynamodbWrapper) Get(keyName string, keyValue string) (itemOutput *dynamodb.GetItemOutput, err error) {
	itemOutput, err = dynamodbSvc.dynamodb.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(dynamodbSvc.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			keyName: {
				S: aws.String(keyValue),
			},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return
	}

	if itemOutput.Item == nil {
		err = ErrDynamoDBKeyNotFound
	}

	return
}

func (dynamodbSvc *DynamodbWrapper) GetBatch(keys []map[string]*dynamodb.AttributeValue) ([]map[string]*dynamodb.AttributeValue, error) {
	input := &dynamodb.BatchGetItemInput{
		RequestItems: map[string]*dynamodb.KeysAndAttributes{
			DynamodbSvc.tableName: {
				ConsistentRead: aws.Bool(true),
				Keys:           keys,
			},
		},
	}

	batchResult, err := DynamodbSvc.dynamodb.BatchGetItem(input)
	if err != nil {
		return nil, err
	}
	results := batchResult.Responses[DynamodbSvc.tableName]

	if batchResult.UnprocessedKeys[dynamodbSvc.tableName] != nil {
		if len(batchResult.UnprocessedKeys[dynamodbSvc.tableName].Keys) > 0 {
			time.Sleep(500 * time.Millisecond)
			retryKeys, err := dynamodbSvc.GetBatch(batchResult.UnprocessedKeys[dynamodbSvc.tableName].Keys)
			if err != nil {
				return nil, err
			}
			results = append(results, retryKeys...)
		}
	}

	return results, nil
}

func (dynamodbSvc *DynamodbWrapper) Set(item interface{}) error {
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		log.Fatalf("got error marshalling dynamodb item: %s", err)
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(dynamodbSvc.tableName),
	}

	_, err = dynamodbSvc.dynamodb.PutItem(input)
	if err != nil {
		log.Fatalf("got error calling PutItem: %s", err)
	}

	return nil
}

func (dynamodbSvc *DynamodbWrapper) SetBatch(request []*dynamodb.WriteRequest) error {
	input := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			dynamodbSvc.tableName: request,
		},
	}

	result, err := dynamodbSvc.dynamodb.BatchWriteItem(input)
	if err != nil {
		return err
	}

	if len(result.UnprocessedItems[dynamodbSvc.tableName]) > 0 {
		time.Sleep(500 * time.Millisecond)
		return dynamodbSvc.SetBatch(result.UnprocessedItems[dynamodbSvc.tableName])
	}

	return nil
}

func (dynamodbSvc *DynamodbWrapper) Update(input dynamodb.UpdateItemInput) error {
	_, err := dynamodbSvc.dynamodb.UpdateItem(&input)
	return err
}

func CreateTable(tagValue, tableName string, dynamodbInstance *dynamodb.DynamoDB) error {
	if IsTableCreated(dynamodbInstance, tableName) {
		return nil
	}

	_, err := dynamodbInstance.CreateTable(&dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("MetadataKey"),
				AttributeType: aws.String(dynamodb.ScalarAttributeTypeS),
			},
		},
		TableName: aws.String(tableName),
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("MetadataKey"),
				KeyType:       aws.String(dynamodb.KeyTypeHash),
			},
		},
		BillingMode: aws.String(dynamodb.BillingModePayPerRequest),
		Tags: []*dynamodb.Tag{
			{
				Key:   aws.String("env"),
				Value: aws.String(tagValue),
			},
		},
	})

	if err != nil {
		return err
	}

	// Wait for table to be created
	created := IsTableCreated(dynamodbInstance, tableName)
	for !created {
		created = IsTableCreated(dynamodbInstance, tableName)
		time.Sleep(1 * time.Second)
	}

	_, err = dynamodbInstance.UpdateTimeToLive(&dynamodb.UpdateTimeToLiveInput{
		TableName: aws.String(tableName),
		TimeToLiveSpecification: &dynamodb.TimeToLiveSpecification{
			AttributeName: aws.String("TTL"),
			Enabled:       aws.Bool(true),
		},
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Message() != "TimeToLive is already enabled" {
				return err
			}
			return nil
		}
	}

	return err
}

func (dynamodbSvc *DynamodbWrapper) DeleteTable() error {
	input := &dynamodb.DeleteTableInput{
		TableName: aws.String(dynamodbSvc.tableName),
	}
	_, err := dynamodbSvc.dynamodb.DeleteTable(input)

	return err
}

func IsTableCreated(dynamodbInstance *dynamodb.DynamoDB, tableName string) (created bool) {
	created = false
	describeTableInput := &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}
	tableOutput, err := dynamodbInstance.DescribeTable(describeTableInput)
	if err != nil {
		return
	}

	if aws.StringValue(tableOutput.Table.TableStatus) == dynamodb.TableStatusActive {
		created = true
	}
	return
}
