package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type Datum struct {
	TestRunId     string `json:"test_run_id"`
	TimeSent      string `json:"time_sent"`
	MessageNumber int    `json:"message_number"`
}

type Output struct {
	TestRunId  string `json:"test_run_id"`
	EventId    string `json:"event_id"`
	Body       string `json:"body"`
	TimeDiffNs int    `json:"time_diff_ns"`
}

func handler(sqsEvent events.SQSEvent) error {
	for _, message := range sqsEvent.Records {
		dataSerialized := []byte(message.Body)
		var datum Datum
		err := json.Unmarshal(dataSerialized, &datum)
		if err != nil {
			fmt.Printf("could not deserialize! %+v\n", err)
			continue
		}
		testRunId := datum.TestRunId
		timeSent, err := time.Parse(time.RFC3339Nano, datum.TimeSent)
		if err != nil {
			fmt.Printf("testRunId %s messageId %s body %s:  can't parse timeSent!\n", testRunId, message.MessageId, string(dataSerialized))
			continue
		}
		timeDiff := time.Now().Sub(timeSent)
		output := Output{
			TestRunId:  testRunId,
			EventId:    message.MessageId,
			Body:       message.Body,
			TimeDiffNs: int(timeDiff.Nanoseconds()),
		}
		outputSerialized, _ := json.Marshal(output)
		fmt.Printf("%s\n", string(outputSerialized))
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
