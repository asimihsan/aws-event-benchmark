package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/google/uuid"
	"os"
	"runtime"
	"strconv"
	"time"
)

var (
	queueUrl         string
	numberOfMessages int
	cfg              aws.Config
)

type Datum struct {
	TestRunId     string `json:"test_run_id"`
	TimeSent      string `json:"time_sent"`
	MessageNumber int    `json:"message_number"`
}

func worker(id int, testRunId string, batchNumbers <-chan int, results chan<- bool) {
	sqsClient := sqs.NewFromConfig(cfg, func(options *sqs.Options) {})
	fmt.Printf("worker id %d start\n", id)
	for batchNumber := range batchNumbers {
		fmt.Printf("worker id %d starting batch %d...\n", id, batchNumber)
		entries := make([]types.SendMessageBatchRequestEntry, 10)
		for j := 0; j < 10; j++ {
			datum := Datum{
				TestRunId:     testRunId,
				TimeSent:      time.Now().Format(time.RFC3339Nano),
				MessageNumber: batchNumber*10 + j,
			}
			serialized, _ := json.Marshal(datum)
			entry := types.SendMessageBatchRequestEntry{
				Id:          aws.String(strconv.Itoa(j)),
				MessageBody: aws.String(string(serialized)),
			}
			entries[j] = entry
		}
		input := sqs.SendMessageBatchInput{
			QueueUrl: aws.String(queueUrl),
			Entries:  entries,
		}
		fmt.Printf("worker id %d sending batch %d...\n", id, batchNumber)
		resp, err := sqsClient.SendMessageBatch(context.TODO(), &input)
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

	fmt.Printf("worker %d done\n", id)
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
	fmt.Printf("testRunId %s done\n", testRunId)
	return nil
}

func main() {
	var err error

	region := os.Getenv("REGION")
	queueUrl = os.Getenv("QUEUE_URL")
	numberOfMessages, err = strconv.Atoi(os.Getenv("NUMBER_OF_MESSAGES"))
	if err != nil {
		panic(err)
	}

	cfg, err = config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithDefaultsMode(aws.DefaultsModeInRegion),
		config.WithRetryMode(aws.RetryModeAdaptive),
		config.WithRetryMaxAttempts(3),
	)
	if err != nil {
		panic(err)
	}

	lambda.Start(handler)
}
