package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/caio/go-tdigest/v4"
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
	aggregation := make(map[string]*tdigest.TDigest)
	now := time.Now()
	timeWindow, err := time.ParseDuration("6h")
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
			timeDiffString := strings.TrimSpace(elems[len(elems)-1])
			timeDiff, err := strconv.ParseFloat(timeDiffString, 64)
			if err != nil {
				panic(err)
			}
			if _, ok := aggregation[testRunId]; !ok {
				t, _ := tdigest.New(tdigest.Compression(10000))
				aggregation[testRunId] = t
			}
			_ = aggregation[testRunId].Add(timeDiff)
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

func main() {
	region = os.Getenv("REGION")
	logGroupName = os.Getenv("CLOUDWATCH_LOGS_LOG_GROUP")

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

	lambda.Start(handler)
}
