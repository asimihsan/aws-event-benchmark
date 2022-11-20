package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"time"
)

type Datum struct {
	TestRunId     string `json:"test_run_id"`
	TimeSent      string `json:"time_sent"`
	MessageNumber int    `json:"message_number"`
}

func handler(event events.KinesisEvent) error {
	for _, record := range event.Records {
		dataSerialized := record.Kinesis.Data
		var datum Datum
		err := json.Unmarshal(dataSerialized, &datum)
		if err != nil {
			fmt.Printf("could not deserialize! %+v\n", err)
		}
		testRunId := datum.TestRunId
		timeSent, err := time.Parse(time.RFC3339Nano, datum.TimeSent)
		if err != nil {
			fmt.Printf("testRunId %s eventId %s body %s:  can't parse timeSent!\n", testRunId, record.EventID, string(dataSerialized))
			continue
		}
		timeDiff := time.Now().Sub(timeSent)
		fmt.Printf("testRunId %s eventId %s body %s timeDiff ms %d\n", testRunId, record.EventID, string(dataSerialized), timeDiff.Milliseconds())
	}
	return nil
}

func main() {
	lambda.Start(handler)
}
