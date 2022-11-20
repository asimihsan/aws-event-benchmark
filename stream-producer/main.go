package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/google/uuid"
	"os"
	"runtime"
	"strconv"
	"time"
)

var (
	streamName       string
	numberOfMessages int
	cfg              aws.Config
)

type Datum struct {
	TestRunId     string `json:"test_run_id"`
	TimeSent      string `json:"time_sent"`
	MessageNumber int    `json:"message_number"`
}

func worker(id int, testRunId string, batchNumbers <-chan int, results chan<- bool) {
	kinesisClient := kinesis.NewFromConfig(cfg, func(o *kinesis.Options) {})
	fmt.Printf("worker id %d start\n", id)
	for batchNumber := range batchNumbers {
		fmt.Printf("worker id %d starting batch %d...\n", id, batchNumber)
		entries := make([]types.PutRecordsRequestEntry, 10)
		for j := 0; j < 10; j++ {
			datum := Datum{
				TestRunId:     testRunId,
				TimeSent:      time.Now().Format(time.RFC3339Nano),
				MessageNumber: batchNumber*10 + j,
			}
			serialized, _ := json.Marshal(datum)
			entry := types.PutRecordsRequestEntry{
				Data:         serialized,
				PartitionKey: aws.String(strconv.Itoa(batchNumber)),
			}
			entries[j] = entry
		}
		fmt.Printf("worker id %d sending batch %d...\n", id, batchNumber)
		resp, err := kinesisClient.PutRecords(context.TODO(), &kinesis.PutRecordsInput{
			Records:    entries,
			StreamName: aws.String(streamName),
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				fmt.Printf("message failed. code: %s, msg: %s\n", aerr.Code(), aerr.Message())
			}
		}
		for _, record := range resp.Records {
			if record.ErrorCode != nil {
				fmt.Printf("message failed. code: %s, msg: %s\n", *record.ErrorCode, *record.ErrorMessage)
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
	streamName = os.Getenv("STREAM_NAME")
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
