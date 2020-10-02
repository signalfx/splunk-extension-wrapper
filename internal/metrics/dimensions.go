package metrics

import (
	"github.com/aws/aws-sdk-go/aws/arn"
	"log"
	"os"
)

const dimShutdownCause = "cause"
const dimRegion = "aws_region"
const dimAccountId = "aws_account_id"
const dimFunctionName = "aws_function_name"
const dimFunctionVersion = "aws_function_version"
const dimArn = "aws_arn"
const dimQualifier = "aws_function_qualifier"
const dimRuntime = "aws_function_runtime"
const dimAwsUniqueId = "AWSUniqueId"

func (emitter MetricEmitter) dims(functionArn string) map[string]string {
	parsedArn, err := arn.Parse(functionArn)

	if err != nil {
		log.Panicf("can't parse ARN: %v\n", functionArn)
	}

	return map[string]string{
		dimRegion:          parsedArn.Region,
		dimAccountId:       parsedArn.AccountID,
		dimFunctionName:    emitter.functionName,
		dimFunctionVersion: emitter.functionVersion,
		dimQualifier:       resourceFromArn(parsedArn).qualifier,
		dimArn:             emitter.arnWithVersion(parsedArn),
		dimRuntime:         os.Getenv(awsExecutionEnv),
		dimAwsUniqueId:     emitter.buildAWSUniqueId(parsedArn),
	}
}
