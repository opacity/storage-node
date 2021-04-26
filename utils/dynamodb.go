package utils

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var ErrDynamodbKeyNotFound = errors.New("key does not exist")

type DynamodbWrapper struct {
	dynamodb  *dynamodb.DynamoDB
	tableName string
}

func NewDynamoDBSession(testOrDebug bool, tableName string, region string, endpoint string) (*DynamodbWrapper, error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	awsConfig := aws.NewConfig()
	tagValue := "prod"
	if testOrDebug {
		awsConfig = awsConfig.WithEndpoint(endpoint).WithLogLevel(aws.LogDebugWithHTTPBody)
		tagValue = "dev"
	}

	dynamodbInstance := dynamodb.New(sess, awsConfig)

	err := CreateTable(tagValue, tableName, dynamodbInstance)

	return &DynamodbWrapper{
		dynamodb:  dynamodbInstance,
		tableName: tableName,
	}, err
}

func (dynamodbSvc *DynamodbWrapper) Get(keyName string, keyValue string) (itemOutput *dynamodb.GetItemOutput, err error) {
	itemOutput, err = dynamodbSvc.dynamodb.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(dynamodbSvc.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			keyName: {
				S: aws.String(keyValue),
			},
		},
	})
	if err != nil {
		return
	}

	if itemOutput.Item == nil {
		err = ErrDynamodbKeyNotFound
	}

	return
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
		retryInput := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				dynamodbSvc.tableName: result.UnprocessedItems[dynamodbSvc.tableName],
			},
		}
		_, err := dynamodbSvc.dynamodb.BatchWriteItem(retryInput)

		return err
	}

	return nil
}

func (dynamodbSvc *DynamodbWrapper) Update(input dynamodb.UpdateItemInput) error {
	_, err := dynamodbSvc.dynamodb.UpdateItem(&input)
	return err
}

func CreateTable(tagValue, tableName string, dynamodbInstance *dynamodb.DynamoDB) error {
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
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(3),
			WriteCapacityUnits: aws.Int64(3),
		},
		Tags: []*dynamodb.Tag{
			{
				Key:   aws.String("env"),
				Value: aws.String(tagValue),
			},
		},
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != dynamodb.ErrCodeResourceInUseException {
				return err
			}
		}
		return nil
	}

	input := &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}

	created := false
	for !created {
		tableOutput, err := dynamodbInstance.DescribeTable(input)
		if err != nil {
			return err
		}

		if aws.StringValue(tableOutput.Table.TableStatus) == dynamodb.TableStatusActive {
			created = true
		}

		time.Sleep(2 * time.Second)
	}

	_, err = dynamodbInstance.UpdateTimeToLive(&dynamodb.UpdateTimeToLiveInput{
		TableName: aws.String(tableName),
		TimeToLiveSpecification: &dynamodb.TimeToLiveSpecification{
			AttributeName: aws.String("TTL"),
			Enabled:       aws.Bool(true),
		},
	})
	if err != nil {
		fmt.Print(err.Error())
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Message() != "TimeToLive is already enabled" {
				return err
			}
		}
	}

	return nil
}

func (dynamodbSvc *DynamodbWrapper) DeleteTable() error {
	input := &dynamodb.DeleteTableInput{
		TableName: aws.String(dynamodbSvc.tableName),
	}
	_, err := dynamodbSvc.dynamodb.DeleteTable(input)

	return err
}
