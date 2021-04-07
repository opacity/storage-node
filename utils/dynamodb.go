package utils

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type dynamodbWrapper struct {
	dynamodb  *dynamodb.DynamoDB
	tableName string
}

func newDynamoDBSession(test bool, tableName string) *dynamodbWrapper {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	awsConfig := aws.NewConfig().WithRegion(Env.AwsRegion).WithEndpoint(Env.AwsDynamoDBEndpoint)

	if test {
		awsConfig = awsConfig.WithLogLevel(aws.LogDebugWithHTTPBody)
	}

	return &dynamodbWrapper{
		dynamodb:  dynamodb.New(sess, awsConfig),
		tableName: tableName,
	}
}

func (dynamodbSvc *dynamodbWrapper) Set(item interface{}) error {

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
