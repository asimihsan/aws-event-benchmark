package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awskinesis"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambdaeventsources"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"path"
)

type EventBenchmarkStackProps struct {
	awscdk.StackProps
}

func EventBenchmarkStack(scope constructs.Construct, id string, props *EventBenchmarkStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	queue := awssqs.NewQueue(stack, jsii.String("InputQueue"), &awssqs.QueueProps{
		VisibilityTimeout: awscdk.Duration_Seconds(jsii.Number(300)),
	})

	queueConsumerLambda := awslambda.NewFunction(stack, jsii.String("QueueConsumerFunction"), &awslambda.FunctionProps{
		Runtime:         awslambda.Runtime_PROVIDED_AL2(),
		MemorySize:      jsii.Number(128),
		Timeout:         awscdk.Duration_Seconds(jsii.Number(15)),
		Handler:         jsii.String("queue-consumer"),
		Architecture:    awslambda.Architecture_ARM_64(),
		Code:            awslambda.Code_FromAsset(jsii.String(path.Join("..", "queue-consumer", "build")), nil),
		InsightsVersion: awslambda.LambdaInsightsVersion_VERSION_1_0_135_0(),
	})

	queueConsumerLambda.AddEventSource(awslambdaeventsources.NewSqsEventSource(queue, &awslambdaeventsources.SqsEventSourceProps{
		BatchSize: jsii.Number(1),
		Enabled:   jsii.Bool(true),
	}))

	queueProducerLambda := awslambda.NewFunction(stack, jsii.String("QueueProducerFunction"), &awslambda.FunctionProps{
		Runtime:         awslambda.Runtime_PROVIDED_AL2(),
		MemorySize:      jsii.Number(4096),
		Timeout:         awscdk.Duration_Minutes(jsii.Number(5)),
		Handler:         jsii.String("queue-producer"),
		Architecture:    awslambda.Architecture_ARM_64(),
		Code:            awslambda.Code_FromAsset(jsii.String(path.Join("..", "queue-producer", "build")), nil),
		InsightsVersion: awslambda.LambdaInsightsVersion_VERSION_1_0_135_0(),
		Environment: &map[string]*string{
			"REGION":             stack.Region(),
			"QUEUE_URL":          queue.QueueUrl(),
			"NUMBER_OF_MESSAGES": jsii.String("10000"),
		},
	})
	queue.GrantSendMessages(queueProducerLambda.Role())

	stream := awskinesis.NewStream(stack, jsii.String("Stream"), &awskinesis.StreamProps{
		RetentionPeriod: awscdk.Duration_Days(jsii.Number(1)),
		StreamMode:      awskinesis.StreamMode_PROVISIONED,
		ShardCount:      jsii.Number(10),
		StreamName:      jsii.String("EventBenchmarkStream"),
	})

	streamConsumer := awskinesis.NewCfnStreamConsumer(stack, jsii.String("StreamConsumer"), &awskinesis.CfnStreamConsumerProps{
		ConsumerName: jsii.String("EventBenchmarkStreamConsumer"),
		StreamArn:    stream.StreamArn(),
	})

	streamConsumerLambda := awslambda.NewFunction(stack, jsii.String("StreamConsumerFunction"), &awslambda.FunctionProps{
		Runtime:         awslambda.Runtime_PROVIDED_AL2(),
		MemorySize:      jsii.Number(128),
		Timeout:         awscdk.Duration_Seconds(jsii.Number(15)),
		Handler:         jsii.String("queue-consumer"),
		Architecture:    awslambda.Architecture_ARM_64(),
		Code:            awslambda.Code_FromAsset(jsii.String(path.Join("..", "stream-consumer", "build")), nil),
		InsightsVersion: awslambda.LambdaInsightsVersion_VERSION_1_0_135_0(),
	})

	streamProducerLambda := awslambda.NewFunction(stack, jsii.String("StreamProducerFunction"), &awslambda.FunctionProps{
		Runtime:         awslambda.Runtime_PROVIDED_AL2(),
		MemorySize:      jsii.Number(4096),
		Timeout:         awscdk.Duration_Minutes(jsii.Number(5)),
		Handler:         jsii.String("queue-producer"),
		Architecture:    awslambda.Architecture_ARM_64(),
		Code:            awslambda.Code_FromAsset(jsii.String(path.Join("..", "stream-producer", "build")), nil),
		InsightsVersion: awslambda.LambdaInsightsVersion_VERSION_1_0_135_0(),
		Environment: &map[string]*string{
			"REGION":             stack.Region(),
			"STREAM_NAME":        stream.StreamName(),
			"NUMBER_OF_MESSAGES": jsii.String("10000"),
		},
	})
	stream.GrantWrite(streamProducerLambda.Role())
	awslambda.NewEventSourceMapping(stack, jsii.String("stream-consumer-mapping"), &awslambda.EventSourceMappingProps{
		EventSourceArn:        streamConsumer.AttrConsumerArn(),
		BatchSize:             jsii.Number(1),
		StartingPosition:      awslambda.StartingPosition_TRIM_HORIZON,
		Enabled:               jsii.Bool(true),
		ParallelizationFactor: jsii.Number(10),
		Target:                streamConsumerLambda,
	})

	stream.GrantRead(streamConsumerLambda.Role())
	resources := []*string{streamConsumer.AttrConsumerArn()}
	actions := []*string{jsii.String("kinesis:SubscribeToShard")}
	streamConsumerLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   &actions,
		Resources: &resources,
	}))

	analyzeTestRunLambda := awslambda.NewFunction(stack, jsii.String("AnalyzeTestRunFunction"), &awslambda.FunctionProps{
		Runtime:         awslambda.Runtime_PROVIDED_AL2(),
		MemorySize:      jsii.Number(128),
		Timeout:         awscdk.Duration_Minutes(jsii.Number(1)),
		Handler:         jsii.String("bootstrap"),
		Architecture:    awslambda.Architecture_ARM_64(),
		Code:            awslambda.Code_FromAsset(jsii.String(path.Join("..", "analyze-test-run", "build")), nil),
		InsightsVersion: awslambda.LambdaInsightsVersion_VERSION_1_0_135_0(),
		Environment: &map[string]*string{
			"REGION":                           stack.Region(),
			"QUEUE_CLOUDWATCH_LOGS_LOG_GROUP":  queueConsumerLambda.LogGroup().LogGroupName(),
			"STREAM_CLOUDWATCH_LOGS_LOG_GROUP": streamConsumerLambda.LogGroup().LogGroupName(),
		},
	})
	queueConsumerLambda.LogGroup().Grant(analyzeTestRunLambda.Role(),
		jsii.String("logs:FilterLogEvents"),
	)
	streamConsumerLambda.LogGroup().Grant(analyzeTestRunLambda.Role(),
		jsii.String("logs:FilterLogEvents"),
	)

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	EventBenchmarkStack(app, "EventBenchmarkStack", &EventBenchmarkStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	// If unspecified, this stack will be "environment-agnostic".
	// Account/Region-dependent features and context lookups will not work, but a
	// single synthesized template can be deployed anywhere.
	//---------------------------------------------------------------------------
	return nil

	// Uncomment if you know exactly what account and region you want to deploy
	// the stack to. This is the recommendation for production stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String("123456789012"),
	//  Region:  jsii.String("us-east-1"),
	// }

	// Uncomment to specialize this stack for the AWS Account and Region that are
	// implied by the current CLI configuration. This is recommended for dev
	// stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
	//  Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	// }
}
