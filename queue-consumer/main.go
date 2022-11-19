package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(sqsEvent events.SQSEvent) error {
	for _, message := range sqsEvent.Records {
		testRunId := *message.MessageAttributes["TestRunId"].StringValue
		timeSentString := *message.MessageAttributes["TimeSent"].StringValue
		timeSent, err := time.Parse(time.RFC3339Nano, timeSentString)
		if err != nil {
			fmt.Printf("testRunId %s messageId %s body %s:  can't parse timeSent!\n", testRunId, message.MessageId, message.Body)
			continue
		}
		timeDiff := time.Now().Sub(timeSent)
		fmt.Printf("testRunId %s messageId %s body %s timeDiff ms %d\n", testRunId, message.MessageId, message.Body, timeDiff.Milliseconds())
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
