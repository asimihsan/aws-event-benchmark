package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/caio/go-tdigest/v4"
	"go.uber.org/ratelimit"
	"os"
	"time"
)

var (
	region               string
	queueLogGroupName    string
	streamLogGroupName   string
	cloudwatchlogsClient *cloudwatchlogs.Client
	cloudformationClient *cloudformation.Client
)

type Output struct {
	TestRunId  string `json:"test_run_id"`
	EventId    string `json:"event_id"`
	Body       string `json:"body"`
	TimeDiffNs int    `json:"time_diff_ns"`
}

func analyze(logGroupName string) error {
	aggregation := make(map[string]*tdigest.TDigest)
	now := time.Now()
	timeWindow, err := time.ParseDuration("6h")
	if err != nil {
		return err
	}
	startTime := now.Add(-timeWindow).UnixMilli()
	paginator := cloudwatchlogs.NewFilterLogEventsPaginator(cloudwatchlogsClient, &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: aws.String(logGroupName),
		StartTime:    aws.Int64(startTime),
		//FilterPattern: aws.String("time_diff_ns"),
	})

	// FilterLogEvents is throttled to 10 TPS outside of us-east-1
	rl := ratelimit.New(10)

	for paginator.HasMorePages() {
		rl.Take()
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return fmt.Errorf("failed to get FilterLogEvents page, %w", err)
		}
		for _, event := range page.Events {
			//fmt.Printf("tick\n")
			var output Output
			err := json.Unmarshal([]byte(*event.Message), &output)
			if err != nil {
				//fmt.Printf("could not deserialize event %s: %+v", *event.Message, err)
				continue
			}
			//fmt.Printf("output: %+v\n", output)
			testRunId := output.TestRunId
			timeDiff := time.Nanosecond * time.Duration(output.TimeDiffNs)
			if _, ok := aggregation[testRunId]; !ok {
				t, _ := tdigest.New(tdigest.Compression(10000))
				aggregation[testRunId] = t
			}
			_ = aggregation[testRunId].Add(float64(timeDiff.Milliseconds()))
		}
	}
	for k, digest := range aggregation {
		fmt.Printf("timeRunId %s, count = %d\n", k, digest.Count())
		fmt.Printf("timeRunId %s, p0 = %.3f\n", k, digest.Quantile(0.0))
		fmt.Printf("timeRunId %s, p50 = %.3f\n", k, digest.Quantile(0.5))
		fmt.Printf("timeRunId %s, p90 = %.3f\n", k, digest.Quantile(0.9))
		fmt.Printf("timeRunId %s, p99 = %.3f\n", k, digest.Quantile(0.99))
		fmt.Printf("timeRunId %s, p100 = %.3f\n", k, digest.Quantile(1.0))
	}

	return nil
}

func handler() error {
	fmt.Printf("handler entry\n")
	for _, logGroupName := range []string{streamLogGroupName, queueLogGroupName} {
		fmt.Printf("analyzing log group %s ...\n", logGroupName)
		err := analyze(logGroupName)
		fmt.Printf("analyzed log group %s\n", logGroupName)
		if err != nil {
			fmt.Printf("error analysing %s: %+v\n", logGroupName, err)
		}
	}

	return nil
}

func main() {
	fmt.Printf("init start\n")

	region = os.Getenv("REGION")
	queueLogGroupName = os.Getenv("QUEUE_CLOUDWATCH_LOGS_LOG_GROUP")
	streamLogGroupName = os.Getenv("STREAM_CLOUDWATCH_LOGS_LOG_GROUP")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithDefaultsMode(aws.DefaultsModeInRegion),
		config.WithRetryMode(aws.RetryModeAdaptive),
		config.WithRetryMaxAttempts(3),
	)
	if err != nil {
		panic(err)
	}

	cloudwatchlogsClient = cloudwatchlogs.NewFromConfig(cfg, func(o *cloudwatchlogs.Options) {
	})

	cloudformationClient = cloudformation.NewFromConfig(cfg, func(o *cloudformation.Options) {
	})

	fmt.Printf("init finished\n")

	lambda.Start(handler)
}
