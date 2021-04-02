package utils

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type sqsWrapper struct {
	sqs *sqs.SQS
}

var sqsSvc *sqsWrapper

func newSqsSession() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	sqsSvc = &sqsWrapper{
		sqs: sqs.New(sess),
	}
}

// Sends a message to AWS SQS
func SendMessage(messageBody string) error {
	sqsUrl, err := getQueueURL()
	if err != nil {
		return err
	}
	_, err = sqsSvc.sqs.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(messageBody),
		QueueUrl:    sqsUrl.QueueUrl,
	})

	if err != nil {
		return err
	}

	return nil
}

func getQueueURL() (*sqs.GetQueueUrlOutput, error) {
	result, err := sqsSvc.sqs.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(Env.AwsSqsQueueName),
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
