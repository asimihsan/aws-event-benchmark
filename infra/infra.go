package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
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
		Runtime:         awslambda.Runtime_GO_1_X(),
		MemorySize:      jsii.Number(128),
		Timeout:         awscdk.Duration_Seconds(jsii.Number(15)),
		Handler:         jsii.String("queue-consumer"),
		Architecture:    awslambda.Architecture_X86_64(),
		Code:            awslambda.Code_FromAsset(jsii.String(path.Join("..", "queue-consumer", "build")), nil),
		InsightsVersion: awslambda.LambdaInsightsVersion_VERSION_1_0_143_0(),
	})

	queueConsumerLambda.AddEventSource(awslambdaeventsources.NewSqsEventSource(queue, &awslambdaeventsources.SqsEventSourceProps{
		BatchSize: jsii.Number(1),
		Enabled:   jsii.Bool(true),
	}))

	queueProducerLambda := awslambda.NewFunction(stack, jsii.String("QueueProducerFunction"), &awslambda.FunctionProps{
		Runtime:         awslambda.Runtime_GO_1_X(),
		MemorySize:      jsii.Number(4096),
		Timeout:         awscdk.Duration_Seconds(jsii.Number(30)),
		Handler:         jsii.String("queue-producer"),
		Architecture:    awslambda.Architecture_X86_64(),
		Code:            awslambda.Code_FromAsset(jsii.String(path.Join("..", "queue-producer", "build")), nil),
		InsightsVersion: awslambda.LambdaInsightsVersion_VERSION_1_0_143_0(),
		Environment: &map[string]*string{
			"QUEUE_URL":          queue.QueueUrl(),
			"NUMBER_OF_MESSAGES": jsii.String("1000"),
		},
	})
	queue.GrantSendMessages(queueProducerLambda.Role())

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
