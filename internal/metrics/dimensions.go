// Copyright Splunk Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"github.com/aws/aws-sdk-go/aws/arn"
	"log"
	"os"
)

const dimShutdownCause = "aws_function_shutdown_cause"
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
