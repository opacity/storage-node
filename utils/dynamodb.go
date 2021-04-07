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

// func (dynamodbSvc *DynamodbWrapper) SetBatch(item []interface{}) error {
// 	dynamodbSvc.dynamodb.BatchWri

// 	return nil
// }
