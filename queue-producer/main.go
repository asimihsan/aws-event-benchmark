package main

import (
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/google/uuid"
	"os"
	"runtime"
	"strconv"
	"time"
)

var (
	queueUrl         string
	numberOfMessages int
	sess             *session.Session
	sqsClient        *sqs.SQS
)

func worker(id int, testRunId string, batchNumbers <-chan int, results chan<- bool) {
	sqsClient = sqs.New(sess)
	fmt.Printf("worker id %d start\n", id)
	for batchNumber := range batchNumbers {
		fmt.Printf("worker id %d starting batch %d...\n", id, batchNumber)
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
				MessageBody: aws.String(fmt.Sprintf("foobar %010d", batchNumber*10+j)),
			}
			entries[j] = &entry
		}
		input := sqs.SendMessageBatchInput{
			QueueUrl: aws.String(queueUrl),
			Entries:  entries,
		}
		fmt.Printf("worker id %d sending batch %d...\n", id, batchNumber)
		resp, err := sqsClient.SendMessageBatch(&input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				fmt.Printf("message failed. code: %s, msg: %s\n", aerr.Code(), aerr.Message())
			}
		}
		if len(resp.Failed) > 0 {
			fmt.Printf("worker id %d some messages failed to send!!\n", id)
			for _, message := range resp.Failed {
				fmt.Printf("**failed id: %s, code: %s, message: %s, sender fault: %+v\n", message.Id, message.Code, message.Message, message.SenderFault)
			}
		}
	}

	fmt.Printf("worker %d done", id)
	results <- true
}

func handler() error {
	testRunId := uuid.NewString()
	numberOfBatches := numberOfMessages / 10
	batchNumbers := make(chan int, numberOfBatches)
	results := make(chan bool, numberOfBatches)
	numWorkers := runtime.NumCPU()
	for w := 1; w <= numWorkers; w++ {
		go worker(w, testRunId, batchNumbers, results)
	}
	for i := 0; i <= numberOfMessages/10; i++ {
		batchNumbers <- i
	}
	close(batchNumbers)

	for i := 0; i < numWorkers; i++ {
		<-results
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

	sess = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	lambda.Start(handler)
}
