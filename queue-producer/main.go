package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/google/uuid"
	"os"
	"strconv"
	"time"
)

var (
	queueUrl         string
	numberOfMessages int
	sqsClient        *sqs.SQS
)

func handler(ctx context.Context) error {
	testRunId := uuid.NewString()
	fmt.Printf("testRunId %s, numberOfMessages %d\n", testRunId, numberOfMessages)
	for i := 0; i <= numberOfMessages/10; i++ {
		fmt.Printf("starting batch %d...\n", i)
		entries := make([]*sqs.SendMessageBatchRequestEntry, 10)
		for j := 0; j < 10; j++ {
			entry := sqs.SendMessageBatchRequestEntry{
				Id: aws.String(strconv.Itoa(j)),
				MessageAttributes: map[string]*sqs.MessageAttributeValue{
					"TestRunId": {
						DataType:    aws.String("String"),
						StringValue: aws.String(testRunId),
					},
					"TimeSent": {
						DataType:    aws.String("String"),
						StringValue: aws.String(time.Now().Format(time.RFC3339Nano)),
					},
				},
				MessageBody: aws.String(fmt.Sprintf("foobar %010d", i*10+j)),
			}
			entries[j] = &entry
		}
		input := sqs.SendMessageBatchInput{
			QueueUrl: aws.String(queueUrl),
			Entries:  entries,
		}
		fmt.Printf("sending batch %d...\n", i)
		resp, err := sqsClient.SendMessageBatch(&input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				fmt.Printf("message failed. code: %s, msg: %s\n", aerr.Code(), aerr.Message())
			}
		}
		if len(resp.Failed) > 0 {
			fmt.Printf("some messages failed to send!!\n")
			for _, message := range resp.Failed {
				fmt.Printf("**failed id: %s, code: %s, message: %s, sender fault: %+v\n", message.Id, message.Code, message.Message, message.SenderFault)
			}
		}
	}
	return nil
}

func main() {
	var err error

	queueUrl = os.Getenv("QUEUE_URL")
	numberOfMessages, err = strconv.Atoi(os.Getenv("NUMBER_OF_MESSAGES"))
	if err != nil {
		panic(err)
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	sqsClient = sqs.New(sess)

	lambda.Start(handler)
}
