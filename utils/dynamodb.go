package utils

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type DynamodbWrapper struct {
	dynamodb  *dynamodb.DynamoDB
	tableName string
}

func NewDynamoDBSession(testOrDebug bool, tableName string, region string, endpoint string) *DynamodbWrapper {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	awsConfig := aws.NewConfig()

	if testOrDebug {
		awsConfig = awsConfig.WithEndpoint(endpoint).WithLogLevel(aws.LogDebugWithHTTPBody)
	}

	return &DynamodbWrapper{
		dynamodb:  dynamodb.New(sess, awsConfig),
		tableName: tableName,
	}
}

func (dynamodbSvc *DynamodbWrapper) Get(keyName string, keyValue string) (*dynamodb.GetItemOutput, error) {
	return dynamodbSvc.dynamodb.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(dynamodbSvc.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			keyName: {
				S: aws.String(keyValue),
			},
		},
	})
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
		log.Fatalf("Got error calling PutItem: %s", err)
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
