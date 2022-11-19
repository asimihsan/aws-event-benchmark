package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"go.uber.org/ratelimit"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	region               string
	logGroupName         string
	cloudwatchlogsClient *cloudwatchlogs.Client
	cloudformationClient *cloudformation.Client
)

func handler() error {
	aggregation := make(map[string][]int)
	now := time.Now()
	timeWindow, err := time.ParseDuration("24h")
	if err != nil {
		return err
	}
	startTime := now.Add(-timeWindow).UnixMilli()
	paginator := cloudwatchlogs.NewFilterLogEventsPaginator(cloudwatchlogsClient, &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:  aws.String(logGroupName),
		StartTime:     aws.Int64(startTime),
		FilterPattern: aws.String("testRunId"),
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
			elems := strings.Split(*event.Message, " ")
			testRunId := elems[1]
			timeDiff, _ := strconv.Atoi(elems[len(elems)-1])
			if _, ok := aggregation[testRunId]; !ok {
				aggregation[testRunId] = make([]int, 0)
			}
			measurements, _ := aggregation[testRunId]
			measurementsNew := append(measurements, timeDiff)
			aggregation[testRunId] = measurementsNew
		}
	}
	for k, vs := range aggregation {
		for _, v := range vs {
			fmt.Printf("%s,%d\n", k, v)
		}
	}

	return nil
}

func main() {
	region = os.Getenv("REGION")
	logGroupName = os.Getenv("CLOUDWATCH_LOGS_LOG_GROUP")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithDefaultsMode(aws.DefaultsModeInRegion),
		config.WithRetryMode(aws.RetryModeAdaptive),
		config.WithRetryMaxAttempts(3),
		//config.WithClientLogMode(aws.LogRetries|aws.LogRequest),
	)
	if err != nil {
		panic(err)
	}

	cloudwatchlogsClient = cloudwatchlogs.NewFromConfig(cfg, func(o *cloudwatchlogs.Options) {
	})

	cloudformationClient = cloudformation.NewFromConfig(cfg, func(o *cloudformation.Options) {
	})

	lambda.Start(handler)
}
